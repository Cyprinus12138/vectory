package engine

import (
	"context"
	"github.com/Cyprinus12138/vectory/internal/cluster"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	"github.com/Cyprinus12138/vectory/pkg"
	"github.com/ghodss/yaml"
	etcd "go.etcd.io/etcd/client/v3"
	"os"
	"sync"
)

type Mode int

const (
	Single Mode = iota + 1
	Cluster
)

type IndexManager struct {
	engineStore    *sync.Map // Key: shardKey Value: Index
	indexManifests *sync.Map // Key: indexName Value: IndexManifest
	mode           Mode

	etcd *etcd.Client
}

var manger *IndexManager

// InitManager TODO error resolving and healthy/unhealthy.
// TODO load index async.
func InitManager(ctx context.Context, clusterMode bool, etcdCli *etcd.Client) (err error) {
	engineStore := &sync.Map{}
	indexManifests := &sync.Map{}

	pkg.SetStatus(pkg.Init)
	defer func(err *error) {
		if *err != nil {
			pkg.SetStatus(pkg.Inactive)
		}
	}(&err)

	var mode Mode
	if clusterMode {
		mode = Cluster
		clusterManger := cluster.GetManger()
		response, err := etcdCli.Get(ctx, config.GetIndexManifestPathPrefix(), etcd.WithPrefix())
		if err != nil {
			return err
		}

		for _, kv := range response.Kvs {
			manifest := &IndexManifest{}
			var index Index

			err = yaml.Unmarshal(kv.Value, manifest)
			if err != nil {
				return err
			}

			err = manifest.Validate()
			if err != nil {

			}

			shards := manifest.GenerateShards()
			for _, shard := range shards {
				if !clusterManger.NeedLoad(shard.ShardKey()) {
					continue
				}
				index, err = NewIndex(ctx, manifest, shard)
				if err != nil {

				}
				engineStore.Store(shard.ShardKey(), index)
			}
			indexManifests.Store(manifest.Meta.Name, manifest)
		}
		manger = &IndexManager{
			engineStore:    engineStore,
			indexManifests: indexManifests,
			mode:           mode,
			etcd:           nil,
		}
		return nil
	} else {
		mode = Single
		indexFiles, err := os.ReadDir(config.GetLocalIndexRootPath())
		if err != nil {
			return err
		}
		for _, file := range indexFiles {
			var (
				rawCfg []byte
				index  Index
			)
			manifest := &IndexManifest{}
			rawCfg, err = os.ReadFile(config.GetLocalIndexPath(file.Name()))
			if err != nil {

			}

			err = yaml.Unmarshal(rawCfg, manifest)
			if err != nil {
				return err
			}

			err = manifest.Validate()
			if err != nil {

			}

			shards := manifest.GenerateShards()
			for _, shard := range shards {
				index, err = NewIndex(ctx, manifest, shard)
				if err != nil {

				}
				engineStore.Store(shard.ShardKey(), index)
			}
			indexManifests.Store(manifest.Meta.Name, manifest)
		}

		manger = &IndexManager{
			engineStore:    engineStore,
			indexManifests: indexManifests,
			mode:           mode,
			etcd:           nil,
		}
		return nil
	}
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

	Routing *cluster.Routing
}

func (i *IndexManager) Search(ctx context.Context, indexName string, x []float32, k int64) (result []SearchResult, err error) {
	log := logger.DefaultLoggerWithCtx(ctx).With(logger.String("indexName", indexName))

	clusterManager := cluster.GetManger()
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
	shards := manifest.GenerateShards()
	result = make([]SearchResult, len(shards))
	for idx, shard := range shards {
		if i.mode == Cluster && !clusterManager.NeedLoad(shard.ShardKey()) {
			var iErr error
			routing, iErr := clusterManager.Route(ctx, shard.ShardKey())
			if iErr != nil {
				log.Error("route shard failed", logger.Err(iErr))
			}
			result[idx] = SearchResult{
				Shard:   shard,
				Error:   iErr,
				Result:  nil,
				Routing: routing,
			}
			continue
		}
		result[idx], err = i.SearchShard(ctx, shard, x, k)
		if err != nil {
			log.Error("search failed", logger.Err(err))
		}
	}
	return result, nil
}

func (i *IndexManager) SearchShard(ctx context.Context, shard Shard, x []float32, k int64) (result SearchResult, err error) {
	log := logger.DefaultLoggerWithCtx(ctx).With(logger.String("shard", shard.ShardKey()))

	result = SearchResult{
		Shard: shard,
	}

	val, ok := i.engineStore.Load(shard.ShardKey())
	if !ok {
		log.Error("get shard not found")
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
