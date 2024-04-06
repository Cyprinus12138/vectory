package engine

import (
	"context"
	"github.com/Cyprinus12138/vectory/internal/cluster"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	"github.com/Cyprinus12138/vectory/pkg"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"go.etcd.io/etcd/api/v3/mvccpb"
	etcd "go.etcd.io/etcd/client/v3"
	"os"
	"sync"
)

type Mode int

const (
	Single Mode = iota + 1
	Cluster
)

func (m *Mode) ToString() string {
	switch *m {
	case Single:
		return "Single"
	case Cluster:
		return "Cluster"
	}
	return "Unknown"
}

type IndexManager struct {
	engineStore    *sync.Map // Key: shardKey Value: Index
	indexManifests *sync.Map // Key: indexName Value: *IndexManifest
	pendingShards  *sync.Map // Key: Shard Value: bool (dummy true)
	listeners      *sync.Map // Key: Target(uniqueShardKey) Value: Listener
	mode           Mode

	etcd *etcd.Client
}

var manger *IndexManager

// InitManager TODO load index async.
func InitManager(ctx context.Context, clusterMode bool, etcdCli *etcd.Client) (err error) {
	manger = &IndexManager{
		engineStore:    &sync.Map{},
		indexManifests: &sync.Map{},
		pendingShards:  &sync.Map{},
		listeners:      &sync.Map{},
	}
	log := logger.DefaultLoggerWithCtx(ctx)
	var (
		mode                             Mode
		total, successShard, indexFailed int
	)
	if clusterMode {
		manger.mode = Cluster
		manger.etcd = etcdCli
		response, err := etcdCli.Get(ctx, config.GetIndexManifestPathPrefix(), etcd.WithPrefix())
		if err != nil {
			log.Error(
				"get manifests failed",
				logger.String("mode", mode.ToString()),
				logger.String("path", config.GetNodeMetaPathPrefix()),
				logger.Err(err),
			)
			return config.ErrGetManifestsFailed
		}

		for _, kv := range response.Kvs {
			log.With(logger.String("manifest_key", string(kv.Key)))
			manifest := &IndexManifest{}

			err = yaml.Unmarshal(kv.Value, manifest)
			if err != nil {
				indexFailed += 1
				log.Error("unmarshal manifest failed", logger.String("content", string(kv.Value)), logger.Err(err))
				continue
			}

			idxTotalShard, idxSuccessShard, err := manger.loadIndex(ctx, manifest)
			if err != nil {
				log.Error("invalid index manifest", logger.Err(err))
				continue
			}
			total += idxTotalShard
			idxSuccessShard += successShard
		}

		// Provide a way for cluster manager to trigger the rebalance process.
		cluster.GetManger().SetRebalanceHook(manger.Rebalance)
	} else {
		manger.mode = Single
		indexFiles, err := os.ReadDir(config.GetLocalIndexRootPath())
		if err != nil {
			log.Error(
				"get manifests failed",
				logger.String("mode", mode.ToString()),
				logger.String("path", config.GetNodeMetaPathPrefix()),
				logger.Err(err),
			)
			return config.ErrGetManifestsFailed
		}

		for _, file := range indexFiles {
			log.With(logger.String("manifest_key", file.Name()))
			var rawCfg []byte

			manifest := &IndexManifest{}
			rawCfg, err = os.ReadFile(config.GetLocalIndexPath(file.Name()))
			if err != nil {
				indexFailed += 1
				log.Error("read local manifest file failed", logger.Err(err))
				continue
			}

			err = yaml.Unmarshal(rawCfg, manifest)
			if err != nil {
				indexFailed += 1
				log.Error("unmarshal manifest failed", logger.String("content", string(rawCfg)))
				continue
			}

			idxTotalShard, idxSuccessShard, err := manger.loadIndex(ctx, manifest)
			if err != nil {
				log.Error("invalid index manifest", logger.Err(err))
				continue
			}
			total += idxTotalShard
			idxSuccessShard += successShard
		}
	}

	manger.SyncCluster(ctx)

	if total > 0 && successShard == 0 {
		return config.ErrNoSuccess
	}
	if total > 0 && successShard < total {
		log.Warn("several shards load failed", logger.Int("total", total), logger.Int("successShard", successShard))
		return config.ErrPartlySuccess
	}
	if indexFailed > 0 {
		log.Warn("several index parse failed", logger.Int("failedIndex", indexFailed))
		return config.ErrPartlySuccess
	}

	return nil
}

func GetManager() *IndexManager {
	return manger
}

type Label struct {
	Distance float32
	Label    int64
}

type SearchResult struct {
	Shard  Shard
	Error  error
	Result []Label

	// ToRoute indicate whether this shard need to be routed to another node
	// for processing in the further step.
	ToRoute bool
}

func (i *IndexManager) loadIndex(ctx context.Context, manifest *IndexManifest) (totalShard, successShard int, err error) {
	log := logger.DefaultLoggerWithCtx(ctx)

	err = manifest.Validate()
	if err != nil {
		log.Error("invalid manifest content", logger.Err(err))
		return 0, 0, err
	}
	clusterManger := cluster.GetManger()
	shards := manifest.GenerateShards()
	for _, shard := range shards {
		if _, loaded := i.engineStore.Load(shard.UniqueShardKey()); (i.mode == Cluster && clusterManger != nil && !clusterManger.NeedLoad(shard.ShardKey())) || loaded {
			continue
		}
		totalShard += 1

		index, err := NewIndex(ctx, manifest, shard)
		if err != nil {
			log.Error(
				"init index shard failed",
				logger.Err(err),
				logger.String("shardKey", shard.ShardKey()),
			)
			i.markShardPending(ctx, shard)
			continue
		}
		i.engineStore.Store(shard.UniqueShardKey(), index)
		successShard += 1
	}
	i.indexManifests.Store(manifest.Meta.Name, manifest)
	return totalShard, successShard, nil
}

func (i *IndexManager) deleteIndex(ctx context.Context, indexName string) {
	log := logger.DefaultLoggerWithCtx(ctx)
	var deleted int
	val, loaded := i.indexManifests.LoadAndDelete(indexName)
	if !loaded {
		log.Warn("no such index loaded", logger.String("indexName", indexName))
		return
	}

	manifest, ok := val.(*IndexManifest)
	if !ok {
		log.Warn("invalid manifest stored", logger.Interface("content", val))
		return
	}

	shards := manifest.GenerateUniqueShards()
	for _, shard := range shards {
		val, loaded := i.engineStore.LoadAndDelete(shard.UniqueShardKey())
		if !loaded {
			log.Debug("shard not loaded in current node", logger.String("shardKey", shard.ShardKey()))
			continue
		}
		index, ok := val.(Index)
		if !ok {
			log.Warn("invalid index engine stored", logger.Interface("content", val))
			continue
		}
		index.Delete()
		deleted += 1
		log.Info("engine has been release", logger.String("shardKey", index.Shard().ShardKey()))
	}
	log.Info("index delete done", logger.Int("deletedShards", deleted))
}

func (i *IndexManager) markShardPending(ctx context.Context, shard Shard) {
	logger.CtxWarn(ctx, "shard marked pending", logger.Interface("shard", shard))
	i.pendingShards.Store(shard, true)
	if pkg.GetStatus() == pkg.Healthy {
		pkg.SetStatus(pkg.Unhealthy)
	}
}

func (i *IndexManager) unmarkShardPending(ctx context.Context, shard Shard) {
	_, ok := i.pendingShards.LoadAndDelete(shard)
	if ok {
		logger.CtxInfo(ctx, "shard unmarked pending", logger.Interface("shard", shard))
		empty := true
		i.pendingShards.Range(func(key, value interface{}) bool {
			empty = false
			return false
		})
		if empty && pkg.GetStatus() == pkg.Unhealthy {
			pkg.SetStatus(pkg.Healthy)
		}
	}

}

func (i *IndexManager) Search(ctx context.Context, indexName string, x []float32, k int64) (result []SearchResult, err error) {
	log := logger.DefaultLoggerWithCtx(ctx).With(logger.String("indexName", indexName))

	val, ok := i.indexManifests.Load(indexName)
	if !ok {
		log.Error("get index not found")
		return nil, config.ErrIndexNotFound
	}
	manifest, ok := val.(*IndexManifest)
	if !ok {
		log.Error("type assertion for index manifest failed", logger.Interface("indexManifest", val))
		return nil, config.ErrTypeAssertion
	}
	shards := manifest.GenerateUniqueShards()
	result = make([]SearchResult, len(shards))
	for idx, shard := range shards {
		result[idx], err = i.SearchShard(ctx, shard, x, k)
		if err != nil && (!errors.Is(err, config.ErrShardNotFound) || i.mode == Single) {
			log.Error("search failed", logger.Err(err))
		}
		if i.mode == Cluster && errors.Is(err, config.ErrShardNotFound) {
			result[idx] = SearchResult{
				Shard:   shard,
				Result:  nil,
				ToRoute: true,
			}
			continue
		}

	}
	return result, nil
}

func (i *IndexManager) SearchShard(ctx context.Context, shard Shard, x []float32, k int64) (result SearchResult, err error) {
	log := logger.DefaultLoggerWithCtx(ctx).With(logger.String("shard", shard.ShardKey()))

	result = SearchResult{
		Shard: shard,

		ToRoute: false,
	}

	val, ok := i.engineStore.Load(shard.UniqueShardKey())
	if !ok {
		log.Debug("get shard not found")
		result.Error = config.ErrShardNotFound
		return result, config.ErrShardNotFound
	}

	index, ok := val.(Index)
	if !ok {
		log.Error("type assertion for index failed", logger.Interface("index", val))
		result.Error = config.ErrTypeAssertion
		return result, config.ErrTypeAssertion
	}
	log.With(logger.Interface("indexMeta", index.Meta()))

	distances, labels, err := index.Search(x, k)
	if err != nil {
		log.Error("search index failed", logger.Err(err))
		result.Error = err
		return result, err
	}
	if len(distances) != len(labels) {
		err = config.ErrOutputDimension
		log.Error("wrong output dimension")
		result.Error = err
		return result, err
	}
	labelResult := make([]Label, 0, len(labels))
	for idx := 0; idx < len(labels); idx++ {
		labelResult = append(labelResult, Label{
			Distance: distances[idx],
			Label:    labels[idx],
		})
	}
	result.Result = labelResult

	return result, nil

}

func (i *IndexManager) SyncCluster(ctx context.Context) {
	if i.mode == Single {
		return
	}

	go func(ctx context.Context) {
		log := logger.DefaultLoggerWithCtx(ctx)

		watch := i.etcd.Watch(ctx, config.GetIndexManifestPathPrefix(), etcd.WithPrefix())
		var watchResp etcd.WatchResponse
		select {
		case watchResp = <-watch:
			if watchResp.Canceled {
				logger.Fatal("watch cluster canceled",
					logger.Err(watchResp.Err()),
				)
				return
			}
			for _, event := range watchResp.Events {
				manifest := &IndexManifest{}

				jErr := yaml.Unmarshal(event.Kv.Value, manifest)
				if jErr != nil {
					log.Error(
						"unmarshal manifest failed",
						logger.String("content", string(event.Kv.Value)),
						logger.Err(jErr),
					)
				}
				jErr = manifest.Validate()
				if jErr != nil {
					log.Error("invalid manifest content", logger.Err(jErr))
					continue
				}
				if event.IsCreate() {
					log.Info("new added index manifest", logger.Interface("meta", manifest.Meta))
					totalShards, successShards, err := i.loadIndex(ctx, manifest)
					if err != nil {
						log.Error("invalid index manifest", logger.Err(err))
						continue
					}
					if successShards < totalShards {
						log.Warn(
							"several new added shard loaded failed",
							logger.Int("total", totalShards),
							logger.Int("success", successShards),
						)
					}
				}
				if event.Type == mvccpb.DELETE {
					log.Info("index deletion detected", logger.Interface("meta", manifest.Meta))
					i.deleteIndex(ctx, manifest.Meta.Name)
				}
				if event.IsModify() {
					log.Warn("modify a loaded index is not current supported and coming soon!")
				}
			}
		}
	}(ctx)
}

// Rebalance should be executed when there is change on the distribution of the shard among the nodes.
// Add new node or remove existing node from the cluster should trigger this process.
func (i *IndexManager) Rebalance(ctx context.Context) (err error) {
	defer i.notifyListeners()

	log := logger.DefaultLoggerWithCtx(ctx)
	var totalShard, successShard, deletedShard int
	pkg.SetStatus(pkg.Rebalancing)
	defer func(err *error) {
		empty := true
		i.pendingShards.Range(func(key, value interface{}) bool {
			empty = false
			return false
		})
		if *err != nil || !empty {
			pkg.SetStatus(pkg.Unhealthy)
			return
		} else {
			pkg.SetStatus(pkg.Healthy)
		}
	}(&err)

	clusterManager := cluster.GetManger()

	i.pendingShards.Range(func(key, value any) bool {
		shard, ok := key.(Shard)
		if !ok {
			return true
		}
		if clusterManager.NeedLoad(shard.ShardKey()) {
			i.unmarkShardPending(ctx, shard)
			deletedShard += 1
		}
		return true
	})

	i.indexManifests.Range(func(key, value any) bool {
		manifest, _ := value.(*IndexManifest)

		shards := manifest.GenerateShards()
		for _, shard := range shards {
			needLoad := clusterManager.NeedLoad(shard.ShardKey())
			val, loaded := i.engineStore.Load(shard.UniqueShardKey())
			if needLoad && !loaded {
				totalShard += 1
				index, err := NewIndex(ctx, manifest, shard)
				if err != nil {
					log.Error(
						"init index shard failed",
						logger.Err(err),
						logger.String("shardKey", shard.ShardKey()),
					)
					i.markShardPending(ctx, shard)
					continue
				}
				i.engineStore.Store(shard.UniqueShardKey(), index)
				successShard += 1
			}
			if !needLoad && loaded {
				deletedShard += 1
				index, _ := val.(Index)
				index.Delete()
				i.engineStore.Delete(shard.UniqueShardKey())
			}
		}
		return true
	})

	if successShard < totalShard {
		log.Warn(
			"rebalance done with several shard loaded failed",
			logger.Int("total", totalShard),
			logger.Int("success", successShard),
			logger.Int("deleted", deletedShard),
		)
	} else {
		log.Info(
			"rebalance done",
			logger.Int("total", totalShard),
			logger.Int("success", successShard),
			logger.Int("deleted", deletedShard),
		)
	}
	return nil
}

func (i *IndexManager) ResolveUniqueShard(uShardKey *Shard) (nodes []*cluster.Routing, err error) {
	v, ok := i.indexManifests.Load(uShardKey.IndexName)
	if !ok {
		logger.Error("no such index loaded", logger.String("indexName", uShardKey.IndexName))
		return nil, config.ErrResolveIndexShard
	}

	manifest, ok := v.(*IndexManifest)
	if !ok {
		logger.Error("assert to IndexManifest failed", logger.Interface("manifest", manifest))
		return nil, config.ErrResolveIndexShard
	}

	if uShardKey.ShardId >= manifest.Meta.Shards {
		logger.Error(
			"no such shard",
			logger.String("indexName", uShardKey.IndexName),
			logger.Int32("desireShardId", uShardKey.ShardId),
			logger.Int32("shardNums", manifest.Meta.Shards),
		)
		return nil, config.ErrResolveIndexShard
	}

	clusterManager := cluster.GetManger()
	replicas := uShardKey.GenerateReplicaKeys(int(manifest.Meta.Replicas))
	for _, replica := range replicas {
		routing, err := clusterManager.Route(context.Background(), replica)
		if err != nil {
			logger.Error("route replica failed", logger.Err(err))
			continue
		}
		nodes = append(nodes, routing)
	}
	return nodes, err
}

// notifyListeners will trigger a ResolverNow action immediately for all existing resolvers (aka Listener here).
func (i *IndexManager) notifyListeners() {
	i.listeners.Range(func(key, value any) bool {
		listener, ok := value.(Listener)
		if !ok {
			return true
		}
		listener.Update()
		return true
	})
}

func (i *IndexManager) UnregisterListener(key string) {
	i.listeners.Delete(key)
}

func (i *IndexManager) RegisterListener(key string, listener Listener) {
	i.listeners.Store(key, listener)
}
