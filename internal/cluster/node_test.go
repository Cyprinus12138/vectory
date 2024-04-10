package cluster

import (
	"context"
	"encoding/json"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/utils"
	"github.com/Cyprinus12138/vectory/mocks"
	"github.com/Cyprinus12138/vectory/pkg"
	"github.com/pkg/errors"
	"github.com/serialx/hashring"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	etcd "go.etcd.io/etcd/client/v3"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"
)

var (
	etcdCli   *etcd.Client
	initField fields
	lis       net.Listener
	conf      *config.ClusterMetaConfig
)

func init() {
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

	conf = &config.ClusterMetaConfig{
		ClusterName: "cluster_unittest",
		ClusterMode: config.ClusterSetting{
			Enabled:     true,
			GracePeriod: 15,
			Endpoints:   etcdAddrs,
			Ttl:         5,
			LBMode:      "none",
		},
	}

	initField = fields{
		ctx:         context.Background(),
		etcd:        etcdCli,
		conf:        conf,
		nodeId:      "1233211234567",
		nodeMateMap: &sync.Map{},
		keepaliveLease: &etcd.LeaseGrantResponse{
			ID:  0,
			TTL: 0,
		},
	}

	lis, err = net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
}

func genField() fields {
	return fields{
		ctx:    initField.ctx,
		etcd:   initField.etcd,
		conf:   initField.conf,
		nodeId: utils.GenInstanceId("unit-test"),
		keepaliveLease: &etcd.LeaseGrantResponse{
			ID:  0,
			TTL: 0,
		},
		nodeMateMap: &sync.Map{},
	}
}

type fields struct {
	ctx             context.Context
	etcd            *etcd.Client
	conf            *config.ClusterMetaConfig
	nodeId          string
	keepaliveLease  *etcd.LeaseGrantResponse
	attachedLoad    bool
	clusterHashRing *hashring.HashRing
	nodeMateMap     *sync.Map
	rebalanceHook   func(ctx context.Context) error
}

func (f fields) WithAttachLoad(attach bool) fields {
	f.attachedLoad = attach
	return f
}

func (f fields) WithAllNodeStatus(status pkg.NodeStatus) fields {
	fNew := fields{
		ctx:             f.ctx,
		etcd:            f.etcd,
		conf:            f.conf,
		nodeId:          f.nodeId,
		keepaliveLease:  f.keepaliveLease,
		attachedLoad:    f.attachedLoad,
		clusterHashRing: f.clusterHashRing,
		nodeMateMap:     &sync.Map{},
		rebalanceHook:   f.rebalanceHook,
	}
	f.nodeMateMap.Range(func(key, value any) bool {
		meta := value.(*NodeMeta)
		meta.Status = status
		fNew.nodeMateMap.Store(key, meta)
		return true
	})
	return fNew
}

// WithNodes can be used to substitute the SyncCluster and ReportLoad(with `withLoad` is `true`) in the unit test.
func (f fields) WithNodes(num int, withLoad bool, status pkg.NodeStatus) fields {
	newMap := &sync.Map{}
	nodesIds := make([]string, num)
	for i := 0; i < num; i++ {
		meta := &NodeMeta{
			NodeId: utils.GenInstanceId("unit-test"),
			Addr:   "127.0.0.1:0",
			Status: status,
		}
		newMap.Store(meta.NodeId, meta)
		nodesIds[i] = meta.NodeId
		if withLoad {
			key := config.GetNodeLoadPath(meta.NodeId)
			load := &NodeLoad{
				CPU:     0.1,
				MemFree: 123,
			}
			loadStr, _ := json.Marshal(load)
			_, err := f.etcd.Put(f.ctx, key, string(loadStr))
			if err != nil {
				panic(err)
			}
		}
	}
	f.attachedLoad = withLoad
	f.nodeMateMap = newMap
	f.clusterHashRing = hashring.New(nodesIds)
	return f
}

func (f fields) WithNode(nodeId string, status pkg.NodeStatus) fields {
	meta := &NodeMeta{
		NodeId: nodeId,
		Addr:   "127.0.0.1:0",
		Status: status,
	}
	f.nodeMateMap.Store(meta.NodeId, meta)
	if f.clusterHashRing == nil {
		f.clusterHashRing = hashring.New([]string{meta.NodeId})
	} else {
		f.clusterHashRing = f.clusterHashRing.AddNode(meta.NodeId)
	}
	if f.attachedLoad {
		key := config.GetNodeLoadPath(meta.NodeId)
		load := &NodeLoad{
			CPU:     0.1,
			MemFree: 123,
		}
		loadStr, _ := json.Marshal(load)
		_, err := f.etcd.Put(f.ctx, key, string(loadStr))
		if err != nil {
			panic(err)
		}
	}
	return f
}

func (f fields) WithNodeId(nodeId string) fields {
	f.nodeId = nodeId
	return f
}

func TestEtcdManager_NeedLoad(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &EtcdManager{
				ctx:             tt.fields.ctx,
				etcd:            tt.fields.etcd,
				conf:            tt.fields.conf,
				nodeId:          tt.fields.nodeId,
				keepaliveLease:  tt.fields.keepaliveLease,
				attachedLoad:    tt.fields.attachedLoad,
				clusterHashRing: tt.fields.clusterHashRing,
				nodeMetaMap:     tt.fields.nodeMateMap,
				rebalanceHook:   tt.fields.rebalanceHook,
			}
			if got := e.NeedLoad(tt.args.key); got != tt.want {
				t.Errorf("NeedLoad() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEtcdManager_Register(t *testing.T) {
	type args struct {
		lis net.Listener
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "pass",
			fields:  initField,
			args:    args{lis: lis},
			wantErr: false,
		},
		{
			name:    "pass_1",
			fields:  genField(),
			args:    args{lis: lis},
			wantErr: false,
		},
		{
			name:    "pass_2",
			fields:  genField(),
			args:    args{lis: lis},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &EtcdManager{
				ctx:             tt.fields.ctx,
				etcd:            tt.fields.etcd,
				conf:            tt.fields.conf,
				nodeId:          tt.fields.nodeId,
				keepaliveLease:  tt.fields.keepaliveLease,
				attachedLoad:    tt.fields.attachedLoad,
				clusterHashRing: tt.fields.clusterHashRing,
				nodeMetaMap:     tt.fields.nodeMateMap,
				rebalanceHook:   tt.fields.rebalanceHook,
			}
			if err := e.Register(tt.args.lis); (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}
			if e.keepaliveLease.ID == 0 {
				t.Errorf("no keep alive lease created")
			}
			get, err := e.etcd.Get(e.ctx, config.GetNodeMetaPath(e.nodeId))
			if (err != nil) != tt.wantErr {
				t.Errorf("get node meta failed, error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Logf("got node meta: %v", string(get.Kvs[0].Value))

		})
	}
}

func TestEtcdManager_ReportLoad(t *testing.T) {

	type args struct {
		policy config.LBModeType
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "no report",
			fields: initField,
			args:   args{policy: "none"},
		},
		{
			name:   "report",
			fields: initField,
			args:   args{policy: "cpu"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				load1, load2 string
			)
			e := &EtcdManager{
				ctx:             tt.fields.ctx,
				etcd:            tt.fields.etcd,
				conf:            tt.fields.conf,
				nodeId:          tt.fields.nodeId,
				keepaliveLease:  tt.fields.keepaliveLease,
				attachedLoad:    tt.fields.attachedLoad,
				clusterHashRing: tt.fields.clusterHashRing,
				nodeMetaMap:     tt.fields.nodeMateMap,
				rebalanceHook:   tt.fields.rebalanceHook,
			}
			key := config.GetNodeLoadPath(e.nodeId)
			get1, _ := e.etcd.Get(e.ctx, key)
			e.ReportLoad(tt.args.policy)
			time.Sleep(time.Duration(6) * time.Second)
			get2, _ := e.etcd.Get(e.ctx, key)

			if get1 != nil && get1.Count > 0 {
				load1 = string(get1.Kvs[0].Value)
			}
			if get2 != nil && get2.Count > 0 {
				load2 = string(get2.Kvs[0].Value)
			}
			if (load1 == load2) && (e.attachedLoad) {
				t.Errorf("load not changed as expected")
			}
			t.Logf("policy=%v", tt.args.policy.Standard())
			t.Logf("load1=%s, load2=%s, attachedLoad=%v", load1, load2, e.attachedLoad)
		})
	}
}

func TestEtcdManager_SyncCluster(t *testing.T) {

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "pass one",
			fields:  genField(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &EtcdManager{
				ctx:             tt.fields.ctx,
				etcd:            tt.fields.etcd,
				conf:            tt.fields.conf,
				nodeId:          tt.fields.nodeId,
				keepaliveLease:  tt.fields.keepaliveLease,
				attachedLoad:    tt.fields.attachedLoad,
				clusterHashRing: tt.fields.clusterHashRing,
				nodeMetaMap:     tt.fields.nodeMateMap,
				rebalanceHook:   tt.fields.rebalanceHook,
			}
			if err := e.SyncCluster(); (err != nil) != tt.wantErr {
				t.Errorf("SyncCluster() error = %v, wantErr %v", err, tt.wantErr)
			}
			sizeBefore := e.clusterHashRing.Size()
			err := e.Register(lis)
			if err != nil {
				t.Errorf("register failed: %v", err)
			}

			time.Sleep(time.Duration(100) * time.Millisecond)
			sizeAfter := e.clusterHashRing.Size()
			if sizeAfter != sizeBefore {
				t.Logf("hashring size %d -> %d", sizeBefore, sizeAfter)
			} else {
				t.Errorf("hashring size didn't change: %d", sizeAfter)
			}
		})
	}
}

func TestEtcdManager_Unregister(t *testing.T) {

	tests := []struct {
		name   string
		fields fields
	}{
		{
			name:   "pass",
			fields: initField,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &EtcdManager{
				ctx:             tt.fields.ctx,
				etcd:            tt.fields.etcd,
				conf:            tt.fields.conf,
				nodeId:          tt.fields.nodeId,
				keepaliveLease:  tt.fields.keepaliveLease,
				attachedLoad:    tt.fields.attachedLoad,
				clusterHashRing: tt.fields.clusterHashRing,
				nodeMetaMap:     tt.fields.nodeMateMap,
				rebalanceHook:   tt.fields.rebalanceHook,
			}
			get, err := e.etcd.Get(e.ctx, config.GetNodeMetaPath(e.nodeId))
			if err != nil {
				t.Errorf("get node meta before unregister failed, error = %v", err)
				return
			}
			e.Unregister()
			get, err = e.etcd.Get(e.ctx, config.GetNodeMetaPath(e.nodeId))
			if err == nil {
				t.Errorf("still got node meta: %v", string(get.Kvs[0].Value))
				return
			}
			if !errors.Is(err, rpctypes.ErrKeyNotFound) {
				t.Errorf("got unexpected error: %v", err)
			}
		})
	}
}

func TestEtcdManager_Route(t *testing.T) {

	type args struct {
		ctx context.Context
		key string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "route available",
			fields: initField.WithNodes(3, false, pkg.Healthy),
			args: args{
				ctx: context.Background(),
				key: utils.GenInstanceId("indexName"),
			},
			wantErr: false,
		},
		{
			name:   "route unavailable",
			fields: initField.WithNodes(3, false, pkg.Inactive),
			args: args{
				ctx: context.Background(),
				key: utils.GenInstanceId("indexName"),
			},
			wantErr: true,
		},
		{
			name:   "route available with load",
			fields: initField.WithNodes(3, true, pkg.Healthy).WithAttachLoad(true),
			args: args{
				ctx: context.Background(),
				key: utils.GenInstanceId("indexName"),
			},
			wantErr: false,
		},
		{
			name:   "route available without load",
			fields: initField.WithNodes(3, false, pkg.Healthy).WithAttachLoad(true),
			args: args{
				ctx: context.Background(),
				key: utils.GenInstanceId("indexName"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &EtcdManager{
				ctx:             tt.fields.ctx,
				etcd:            tt.fields.etcd,
				conf:            tt.fields.conf,
				nodeId:          tt.fields.nodeId,
				keepaliveLease:  tt.fields.keepaliveLease,
				attachedLoad:    tt.fields.attachedLoad,
				clusterHashRing: tt.fields.clusterHashRing,
				nodeMetaMap:     tt.fields.nodeMateMap,
				rebalanceHook:   tt.fields.rebalanceHook,
			}

			gotRouting, err := e.Route(tt.args.ctx, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Route() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotRouting == nil {
				return
			}
			tarNode := gotRouting.Node.NodeId

			tarManager := e
			tarManager.nodeId = tarNode
			if !tarManager.NeedLoad(tt.args.key) {
				t.Errorf("not align with the result of NeedResult")
			}

		})
	}
}

func TestEtcdManager_getLoad(t *testing.T) {

	type args struct {
		ctx    context.Context
		nodeId string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantLoad *NodeLoad
		wantErr  bool
	}{
		{
			name:   "pass",
			fields: initField.WithNodes(3, true, pkg.Healthy).WithNode("must", pkg.Healthy),
			args: args{
				ctx:    context.Background(),
				nodeId: "must",
			},
			wantLoad: &NodeLoad{
				CPU:     0.1,
				MemFree: 123,
			},
			wantErr: false,
		},
		{
			name:   "no node",
			fields: initField.WithNodes(3, true, pkg.Healthy),
			args: args{
				ctx:    context.Background(),
				nodeId: "no data",
			},
			wantLoad: nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &EtcdManager{
				ctx:             tt.fields.ctx,
				etcd:            tt.fields.etcd,
				conf:            tt.fields.conf,
				nodeId:          tt.fields.nodeId,
				keepaliveLease:  tt.fields.keepaliveLease,
				attachedLoad:    tt.fields.attachedLoad,
				clusterHashRing: tt.fields.clusterHashRing,
				nodeMetaMap:     tt.fields.nodeMateMap,
				rebalanceHook:   tt.fields.rebalanceHook,
			}
			gotLoad, err := e.getLoad(tt.args.ctx, tt.args.nodeId)
			if (err != nil) != tt.wantErr {
				t.Errorf("getLoad() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotLoad, tt.wantLoad) {
				t.Errorf("getLoad() gotLoad = %v, want %v", gotLoad, tt.wantLoad)
			}
		})
	}
}

func TestEtcdManager_getNode(t *testing.T) {

	type args struct {
		ctx context.Context
		key string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantNode *NodeMeta
		wantErr  bool
	}{
		{
			name:   "health",
			fields: initField.WithNode("health", pkg.Healthy),
			args: args{
				ctx: context.Background(),
				key: "health",
			},
			wantNode: &NodeMeta{
				NodeId: "health",
				Addr:   "127.0.0.1:0",
				Status: pkg.Healthy,
			},
			wantErr: false,
		},
		{
			name:   "no_node",
			fields: initField.WithNodes(0, false, pkg.Healthy),
			args: args{
				ctx: context.Background(),
				key: "no_node",
			},
			wantNode: nil,
			wantErr:  true,
		},
		{
			name:   "nil_ring",
			fields: initField,
			args: args{
				ctx: context.Background(),
				key: "nil_ring",
			},
			wantNode: nil,
			wantErr:  true,
		},
		{
			name:   "rebalance",
			fields: initField.WithNode("rebalance", pkg.Rebalancing),
			args: args{
				ctx: context.Background(),
				key: "rebalance",
			},
			wantNode: &NodeMeta{
				NodeId: "rebalance",
				Addr:   "127.0.0.1:0",
				Status: pkg.Rebalancing,
			},
			wantErr: false,
		},
		{
			name:   "inactive",
			fields: initField.WithNode("inactive", pkg.Inactive),
			args: args{
				ctx: context.Background(),
				key: "inactive",
			},
			wantNode: &NodeMeta{
				NodeId: "inactive",
				Addr:   "127.0.0.1:0",
				Status: pkg.Inactive,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &EtcdManager{
				ctx:             tt.fields.ctx,
				etcd:            tt.fields.etcd,
				conf:            tt.fields.conf,
				nodeId:          tt.fields.nodeId,
				keepaliveLease:  tt.fields.keepaliveLease,
				attachedLoad:    tt.fields.attachedLoad,
				clusterHashRing: tt.fields.clusterHashRing,
				nodeMetaMap:     tt.fields.nodeMateMap,
				rebalanceHook:   tt.fields.rebalanceHook,
			}
			gotNode, err := e.getNode(tt.args.ctx, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("getNode() = %v  error = %v, wantErr %v", gotNode, err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotNode, tt.wantNode) {
				t.Errorf("getNode() gotNode = %v, want %v", gotNode, tt.wantNode)
			}
			t.Logf("getNode = %v", gotNode)
		})
	}
}

func TestEtcdManager_getNodeMeta(t *testing.T) {

	type args struct {
		ctx    context.Context
		nodeId string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *NodeMeta
		wantErr bool
	}{
		{
			name:   "pass",
			fields: initField.WithNode("pass", pkg.Healthy),
			args: args{
				ctx:    context.Background(),
				nodeId: "pass",
			},
			want: &NodeMeta{
				NodeId: "pass",
				Addr:   "127.0.0.1:0",
				Status: pkg.Healthy,
			},
			wantErr: false,
		},
		{
			name:   "not_found",
			fields: initField,
			args: args{
				ctx:    context.Background(),
				nodeId: "not_found",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &EtcdManager{
				ctx:             tt.fields.ctx,
				etcd:            tt.fields.etcd,
				conf:            tt.fields.conf,
				nodeId:          tt.fields.nodeId,
				keepaliveLease:  tt.fields.keepaliveLease,
				attachedLoad:    tt.fields.attachedLoad,
				clusterHashRing: tt.fields.clusterHashRing,
				nodeMetaMap:     tt.fields.nodeMateMap,
				rebalanceHook:   tt.fields.rebalanceHook,
			}
			got, err := e.getNodeMeta(tt.args.ctx, tt.args.nodeId)
			if (err != nil) != tt.wantErr {
				t.Errorf("getNodeMeta() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getNodeMeta() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEtcdManager_setNodeStatus(t *testing.T) {

	type args struct {
		ctx    context.Context
		status pkg.NodeStatus
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "set_inactive_healthy",
			fields: initField.WithNode("health", pkg.Inactive).WithNodeId("health"),
			args: args{
				ctx:    context.Background(),
				status: pkg.Healthy,
			},
		},
		{
			name:   "set_health_inactive",
			fields: initField.WithNode("inactive", pkg.Healthy).WithNodeId("inactive"),
			args: args{
				ctx:    context.Background(),
				status: pkg.Inactive,
			},
		},
		{
			name:   "set_healthy_healthy",
			fields: initField.WithNode("health", pkg.Healthy).WithNodeId("health"),
			args: args{
				ctx:    context.Background(),
				status: pkg.Healthy,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &EtcdManager{
				ctx:             tt.fields.ctx,
				etcd:            tt.fields.etcd,
				conf:            tt.fields.conf,
				nodeId:          tt.fields.nodeId,
				keepaliveLease:  tt.fields.keepaliveLease,
				attachedLoad:    tt.fields.attachedLoad,
				clusterHashRing: tt.fields.clusterHashRing,
				nodeMetaMap:     tt.fields.nodeMateMap,
				rebalanceHook:   tt.fields.rebalanceHook,
			}
			err := e.Register(lis)
			if err != nil {
				t.Errorf("register failed: %v", err)
				return
			}
			err = e.SyncCluster()
			if err != nil {
				t.Errorf("snyc cluster failed: %v", err)
				return
			}

			e.setNodeStatus(tt.args.ctx, tt.args.status)

			meta, err := e.getNodeMeta(e.ctx, e.nodeId)
			if err != nil {
				t.Errorf("get node meta failed: %v", err)
				return
			}
			if meta.Status != tt.args.status {
				t.Errorf("expaectd: %v, got: %v", tt.args.status, meta.Status)
				return
			}
		})
	}
}

func TestInitEtcdManager(t *testing.T) {
	type args struct {
		ctx    context.Context
		e      *etcd.Client
		conf   *config.ClusterMetaConfig
		nodeId string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "pass",
			args: args{
				ctx:    context.Background(),
				e:      etcdCli,
				conf:   conf,
				nodeId: "pass",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitEtcdManager(tt.args.ctx, tt.args.e, tt.args.conf, tt.args.nodeId)
		})
	}
}

func TestRouting_Weight(t *testing.T) {
	type fields struct {
		Node *NodeMeta
		Load *NodeLoad
	}
	tests := []struct {
		name   string
		fields fields
		want   uint32
	}{
		{
			name: "normal",
			fields: fields{
				Node: &NodeMeta{
					NodeId: "normal",
					Addr:   "123",
					Status: pkg.Healthy,
				},
				Load: &NodeLoad{
					CPU:     0.10,
					MemFree: 123,
				},
			},
			want: 900,
		},
		{
			name: "no_load",
			fields: fields{
				Node: &NodeMeta{
					NodeId: "normal",
					Addr:   "123",
					Status: pkg.Healthy,
				},
				Load: nil,
			},
			want: 1000,
		},
		{
			name: "no_cpu",
			fields: fields{
				Node: &NodeMeta{
					NodeId: "normal",
					Addr:   "123",
					Status: pkg.Healthy,
				},
				Load: &NodeLoad{
					CPU:     0.0,
					MemFree: 123,
				},
			},
			want: 1000,
		},
		{
			name: "invalid",
			fields: fields{
				Node: &NodeMeta{
					NodeId: "invalid",
					Addr:   "123",
					Status: pkg.Inactive,
				},
				Load: &NodeLoad{
					CPU:     0.10,
					MemFree: 123,
				},
			},
			want: 0,
		},
		{
			name: "unhealthy",
			fields: fields{
				Node: &NodeMeta{
					NodeId: "unhealthy",
					Addr:   "123",
					Status: pkg.Unhealthy,
				},
				Load: &NodeLoad{
					CPU:     0.10,
					MemFree: 123,
				},
			},
			want: 90,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Routing{
				Node: tt.fields.Node,
				Load: tt.fields.Load,
			}
			if got := r.Weight(); got != tt.want {
				t.Errorf("Weight() = %v, want %v", got, tt.want)
			}
		})
	}
}
