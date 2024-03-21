package engine

import (
	"context"
	"github.com/Cyprinus12138/vectory/internal/cluster"
	"github.com/Cyprinus12138/vectory/internal/config"
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
	engineStore    *sync.Map
	indexManifests *sync.Map
	mode           Mode

	etcd *etcd.Client
}

func NewManager(ctx context.Context, clusterMode bool, etcdCli *etcd.Client) (*IndexManager, error) {
	engineStore := &sync.Map{}
	indexManifests := &sync.Map{}

	var mode Mode
	if clusterMode {
		mode = Cluster
		clusterManger := cluster.GetManger()
		response, err := etcdCli.Get(ctx, config.GetIndexManifestPathPrefix(), etcd.WithPrefix())
		if err != nil {
			return nil, err
		}

		for _, kv := range response.Kvs {
			manifest := &IndexManifest{}
			var index Index

			err = yaml.Unmarshal(kv.Value, manifest)
			if err != nil {
				return nil, err
			}

			err = manifest.Validate()
			if err != nil {

			}

			shards := manifest.GenerateShards()
			for _, shard := range shards {
				if !clusterManger.NeedLoad(GetShardKey(*manifest, shard)) {
					continue
				}
				index, err = NewIndex(ctx, manifest, shard)
				if err != nil {

				}
				indexManifests.Store(index.ShardKey(), manifest)
				engineStore.Store(index.ShardKey(), index)
			}

		}

		return &IndexManager{
			engineStore:    engineStore,
			indexManifests: indexManifests,
			mode:           mode,
			etcd:           nil,
		}, nil

	} else {
		mode = Single
		indexFiles, err := os.ReadDir(config.GetLocalIndexRootPath())
		if err != nil {
			return nil, err
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
				return nil, err
			}

			err = manifest.Validate()
			if err != nil {

			}

			shards := manifest.GenerateShards()
			for _, shard := range shards {
				index, err = NewIndex(ctx, manifest, shard)
				if err != nil {

				}
				indexManifests.Store(index.ShardKey(), manifest)
				engineStore.Store(index.ShardKey(), index)
			}
		}
		return &IndexManager{
			engineStore:    engineStore,
			indexManifests: indexManifests,
			mode:           mode,
			etcd:           nil,
		}, nil
	}
}
