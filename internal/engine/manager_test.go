package engine

import (
	"context"
	"github.com/Cyprinus12138/vectory/internal/cluster"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/utils/config_manager"
	"github.com/Cyprinus12138/vectory/mocks"
	"github.com/spf13/viper"
	etcd "go.etcd.io/etcd/client/v3"
	"os"
	"reflect"
	"sync"
	"testing"
)

var (
	etcdCli *etcd.Client
)

func init() {
	err := config_manager.Init("../../tests/config.yml", []viper.RegisteredConfig{
		{
			Key:      "env",
			CanBeNil: true,
		},
	})
	if err != nil {
		panic(err)
	}

	etcdSvrs, err := mocks.StartMockServers(1)
	if err != nil {
		panic(err)
	}

	etcdAddrs := make([]string, 1)
	for i, svr := range etcdSvrs.Servers {
		etcdAddrs[i] = svr.ResolverAddress().Addr
	}

	etcdCli, err = etcd.NewFromURLs(etcdAddrs)
	if err != nil {
		panic(err)
	}

	err = os.Setenv(config.EnvIndexPath, "../../tests/index/manager_init_success")
	if err != nil {
		panic(err)
	}

	indexFiles, err := os.ReadDir(config.GetLocalIndexRootPath())
	if err != nil {
		panic(err)
	}

	for _, file := range indexFiles {
		rawCfg, err := os.ReadFile(config.GetLocalIndexPath(file.Name()))
		if err != nil {
			panic(err)
		}

		_, err = etcdCli.Put(context.Background(), file.Name(), string(rawCfg))
		if err != nil {
			panic(err)
		}
	}

	mocks.InitMockCluster(context.Background(), etcdCli, 3)
}

func TestIndexManager_Rebalance(t *testing.T) {
	type fields struct {
		engineStore    *sync.Map
		indexManifests *sync.Map
		pendingShards  *sync.Map
		listeners      *sync.Map
		mode           Mode
		etcd           *etcd.Client
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &IndexManager{
				engineStore:    tt.fields.engineStore,
				indexManifests: tt.fields.indexManifests,
				pendingShards:  tt.fields.pendingShards,
				listeners:      tt.fields.listeners,
				mode:           tt.fields.mode,
				etcd:           tt.fields.etcd,
			}
			if err := i.Rebalance(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Rebalance() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIndexManager_RegisterListener(t *testing.T) {
	type fields struct {
		engineStore    *sync.Map
		indexManifests *sync.Map
		pendingShards  *sync.Map
		listeners      *sync.Map
		mode           Mode
		etcd           *etcd.Client
	}
	type args struct {
		key      string
		listener Listener
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &IndexManager{
				engineStore:    tt.fields.engineStore,
				indexManifests: tt.fields.indexManifests,
				pendingShards:  tt.fields.pendingShards,
				listeners:      tt.fields.listeners,
				mode:           tt.fields.mode,
				etcd:           tt.fields.etcd,
			}
			i.RegisterListener(tt.args.key, tt.args.listener)
		})
	}
}

func TestIndexManager_ResolveUniqueShard(t *testing.T) {
	type fields struct {
		engineStore    *sync.Map
		indexManifests *sync.Map
		pendingShards  *sync.Map
		listeners      *sync.Map
		mode           Mode
		etcd           *etcd.Client
	}
	type args struct {
		uShardKey *Shard
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantNodes []*cluster.Routing
		wantErr   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &IndexManager{
				engineStore:    tt.fields.engineStore,
				indexManifests: tt.fields.indexManifests,
				pendingShards:  tt.fields.pendingShards,
				listeners:      tt.fields.listeners,
				mode:           tt.fields.mode,
				etcd:           tt.fields.etcd,
			}
			gotNodes, err := i.ResolveUniqueShard(tt.args.uShardKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveUniqueShard() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotNodes, tt.wantNodes) {
				t.Errorf("ResolveUniqueShard() gotNodes = %v, want %v", gotNodes, tt.wantNodes)
			}
		})
	}
}

func TestIndexManager_Search(t *testing.T) {
	type fields struct {
		engineStore    *sync.Map
		indexManifests *sync.Map
		pendingShards  *sync.Map
		listeners      *sync.Map
		mode           Mode
		etcd           *etcd.Client
	}
	type args struct {
		ctx       context.Context
		indexName string
		x         []float32
		k         int64
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantResult []SearchResult
		wantErr    bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &IndexManager{
				engineStore:    tt.fields.engineStore,
				indexManifests: tt.fields.indexManifests,
				pendingShards:  tt.fields.pendingShards,
				listeners:      tt.fields.listeners,
				mode:           tt.fields.mode,
				etcd:           tt.fields.etcd,
			}
			gotResult, err := i.Search(tt.args.ctx, tt.args.indexName, tt.args.x, tt.args.k)
			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("Search() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}

func TestIndexManager_SearchShard(t *testing.T) {
	type fields struct {
		engineStore    *sync.Map
		indexManifests *sync.Map
		pendingShards  *sync.Map
		listeners      *sync.Map
		mode           Mode
		etcd           *etcd.Client
	}
	type args struct {
		ctx   context.Context
		shard Shard
		x     []float32
		k     int64
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantResult SearchResult
		wantErr    bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &IndexManager{
				engineStore:    tt.fields.engineStore,
				indexManifests: tt.fields.indexManifests,
				pendingShards:  tt.fields.pendingShards,
				listeners:      tt.fields.listeners,
				mode:           tt.fields.mode,
				etcd:           tt.fields.etcd,
			}
			gotResult, err := i.SearchShard(tt.args.ctx, tt.args.shard, tt.args.x, tt.args.k)
			if (err != nil) != tt.wantErr {
				t.Errorf("SearchShard() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("SearchShard() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}

func TestIndexManager_SyncCluster(t *testing.T) {
	type fields struct {
		engineStore    *sync.Map
		indexManifests *sync.Map
		pendingShards  *sync.Map
		listeners      *sync.Map
		mode           Mode
		etcd           *etcd.Client
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &IndexManager{
				engineStore:    tt.fields.engineStore,
				indexManifests: tt.fields.indexManifests,
				pendingShards:  tt.fields.pendingShards,
				listeners:      tt.fields.listeners,
				mode:           tt.fields.mode,
				etcd:           tt.fields.etcd,
			}
			i.SyncCluster(tt.args.ctx)
		})
	}
}

func TestIndexManager_UnregisterListener(t *testing.T) {
	type fields struct {
		engineStore    *sync.Map
		indexManifests *sync.Map
		pendingShards  *sync.Map
		listeners      *sync.Map
		mode           Mode
		etcd           *etcd.Client
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &IndexManager{
				engineStore:    tt.fields.engineStore,
				indexManifests: tt.fields.indexManifests,
				pendingShards:  tt.fields.pendingShards,
				listeners:      tt.fields.listeners,
				mode:           tt.fields.mode,
				etcd:           tt.fields.etcd,
			}
			i.UnregisterListener(tt.args.key)
		})
	}
}

func TestIndexManager_deleteIndex(t *testing.T) {
	type fields struct {
		engineStore    *sync.Map
		indexManifests *sync.Map
		pendingShards  *sync.Map
		listeners      *sync.Map
		mode           Mode
		etcd           *etcd.Client
	}
	type args struct {
		ctx       context.Context
		indexName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &IndexManager{
				engineStore:    tt.fields.engineStore,
				indexManifests: tt.fields.indexManifests,
				pendingShards:  tt.fields.pendingShards,
				listeners:      tt.fields.listeners,
				mode:           tt.fields.mode,
				etcd:           tt.fields.etcd,
			}
			i.deleteIndex(tt.args.ctx, tt.args.indexName)
		})
	}
}

func TestIndexManager_loadIndex(t *testing.T) {
	type fields struct {
		engineStore    *sync.Map
		indexManifests *sync.Map
		pendingShards  *sync.Map
		listeners      *sync.Map
		mode           Mode
		etcd           *etcd.Client
	}
	type args struct {
		ctx      context.Context
		manifest *IndexManifest
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		wantTotalShard   int
		wantSuccessShard int
		wantErr          bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &IndexManager{
				engineStore:    tt.fields.engineStore,
				indexManifests: tt.fields.indexManifests,
				pendingShards:  tt.fields.pendingShards,
				listeners:      tt.fields.listeners,
				mode:           tt.fields.mode,
				etcd:           tt.fields.etcd,
			}
			gotTotalShard, gotSuccessShard, err := i.loadIndex(tt.args.ctx, tt.args.manifest)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotTotalShard != tt.wantTotalShard {
				t.Errorf("loadIndex() gotTotalShard = %v, want %v", gotTotalShard, tt.wantTotalShard)
			}
			if gotSuccessShard != tt.wantSuccessShard {
				t.Errorf("loadIndex() gotSuccessShard = %v, want %v", gotSuccessShard, tt.wantSuccessShard)
			}
		})
	}
}

func TestIndexManager_markShardPending(t *testing.T) {
	type fields struct {
		engineStore    *sync.Map
		indexManifests *sync.Map
		pendingShards  *sync.Map
		listeners      *sync.Map
		mode           Mode
		etcd           *etcd.Client
	}
	type args struct {
		ctx   context.Context
		shard Shard
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &IndexManager{
				engineStore:    tt.fields.engineStore,
				indexManifests: tt.fields.indexManifests,
				pendingShards:  tt.fields.pendingShards,
				listeners:      tt.fields.listeners,
				mode:           tt.fields.mode,
				etcd:           tt.fields.etcd,
			}
			i.markShardPending(tt.args.ctx, tt.args.shard)
		})
	}
}

func TestIndexManager_notifyListeners(t *testing.T) {
	type fields struct {
		engineStore    *sync.Map
		indexManifests *sync.Map
		pendingShards  *sync.Map
		listeners      *sync.Map
		mode           Mode
		etcd           *etcd.Client
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &IndexManager{
				engineStore:    tt.fields.engineStore,
				indexManifests: tt.fields.indexManifests,
				pendingShards:  tt.fields.pendingShards,
				listeners:      tt.fields.listeners,
				mode:           tt.fields.mode,
				etcd:           tt.fields.etcd,
			}
			i.notifyListeners()
		})
	}
}

func TestIndexManager_unmarkShardPending(t *testing.T) {
	type fields struct {
		engineStore    *sync.Map
		indexManifests *sync.Map
		pendingShards  *sync.Map
		listeners      *sync.Map
		mode           Mode
		etcd           *etcd.Client
	}
	type args struct {
		ctx   context.Context
		shard Shard
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &IndexManager{
				engineStore:    tt.fields.engineStore,
				indexManifests: tt.fields.indexManifests,
				pendingShards:  tt.fields.pendingShards,
				listeners:      tt.fields.listeners,
				mode:           tt.fields.mode,
				etcd:           tt.fields.etcd,
			}
			i.unmarkShardPending(tt.args.ctx, tt.args.shard)
		})
	}
}

func TestInitManager(t *testing.T) {
	type args struct {
		ctx         context.Context
		clusterMode bool
		etcdCli     *etcd.Client
	}
	tests := []struct {
		name     string
		args     args
		wantErr  bool
		indexDir string
	}{
		{
			name: "single_success",
			args: args{
				ctx:         context.Background(),
				clusterMode: false,
				etcdCli:     nil,
			},
			indexDir: "../../tests/index/manager_init_success",
		},
		{
			name: "single_partly_success",
			args: args{
				ctx:         context.Background(),
				clusterMode: false,
				etcdCli:     nil,
			},
			indexDir: "../../tests/index/manager_init_partly",
			wantErr:  true,
		},
		{
			name: "single_failed",
			args: args{
				ctx:         context.Background(),
				clusterMode: false,
				etcdCli:     nil,
			},
			indexDir: "../../tests/index/manager_init_failed",
			wantErr:  true,
		},
		{
			name: "cluster",
			args: args{
				ctx:         context.Background(),
				clusterMode: true,
				etcdCli:     etcdCli,
			},
		},
	}
	for _, tt := range tests {
		err := os.Setenv(config.EnvIndexPath, tt.indexDir)
		if err != nil {
			panic(err)
		}
		t.Run(tt.name, func(t *testing.T) {
			if err := InitManager(tt.args.ctx, tt.args.clusterMode, tt.args.etcdCli); (err != nil) != tt.wantErr {
				t.Errorf("InitManager() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMode_ToString(t *testing.T) {
	tests := []struct {
		name string
		m    Mode
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.ToString(); got != tt.want {
				t.Errorf("ToString() = %v, want %v", got, tt.want)
			}
		})
	}
}
