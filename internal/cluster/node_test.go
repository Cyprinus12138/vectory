package cluster

import (
	"context"
	"github.com/Cyprinus12138/vectory/internal/config"
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

	initField = fields{
		ctx:  context.Background(),
		etcd: etcdCli,
		conf: &config.ClusterMetaConfig{
			ClusterName: "cluster_uniitest",
			ClusterMode: config.ClusterSetting{
				Enabled:     true,
				GracePeriod: 15,
				Endpoints:   etcdAddrs,
				Ttl:         5,
				LBMode:      "none",
			},
		},
		nodeId:      "1233211234567",
		nodeMateMap: sync.Map{},
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

type fields struct {
	ctx             context.Context
	etcd            *etcd.Client
	conf            *config.ClusterMetaConfig
	nodeId          string
	keepaliveLease  *etcd.LeaseGrantResponse
	attachedLoad    bool
	clusterHashRing *hashring.HashRing
	nodeMateMap     sync.Map
	rebalanceHook   func(ctx context.Context) error
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
				nodeMateMap:     tt.fields.nodeMateMap,
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
				nodeMateMap:     tt.fields.nodeMateMap,
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
				nodeMateMap:     tt.fields.nodeMateMap,
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

func TestEtcdManager_Route(t *testing.T) {

	type args struct {
		ctx context.Context
		key string
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantRouting *Routing
		wantErr     bool
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
				nodeMateMap:     tt.fields.nodeMateMap,
				rebalanceHook:   tt.fields.rebalanceHook,
			}
			gotRouting, err := e.Route(tt.args.ctx, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Route() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRouting, tt.wantRouting) {
				t.Errorf("Route() gotRouting = %v, want %v", gotRouting, tt.wantRouting)
			}
		})
	}
}

func TestEtcdManager_SetRebalanceHook(t *testing.T) {

	type args struct {
		f func(ctx context.Context) error
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
			e := &EtcdManager{
				ctx:             tt.fields.ctx,
				etcd:            tt.fields.etcd,
				conf:            tt.fields.conf,
				nodeId:          tt.fields.nodeId,
				keepaliveLease:  tt.fields.keepaliveLease,
				attachedLoad:    tt.fields.attachedLoad,
				clusterHashRing: tt.fields.clusterHashRing,
				nodeMateMap:     tt.fields.nodeMateMap,
				rebalanceHook:   tt.fields.rebalanceHook,
			}
			e.SetRebalanceHook(tt.args.f)
		})
	}
}

func TestEtcdManager_SyncCluster(t *testing.T) {

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
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
				nodeMateMap:     tt.fields.nodeMateMap,
				rebalanceHook:   tt.fields.rebalanceHook,
			}
			if err := e.SyncCluster(); (err != nil) != tt.wantErr {
				t.Errorf("SyncCluster() error = %v, wantErr %v", err, tt.wantErr)
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
				nodeMateMap:     tt.fields.nodeMateMap,
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
				nodeMateMap:     tt.fields.nodeMateMap,
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
				nodeMateMap:     tt.fields.nodeMateMap,
				rebalanceHook:   tt.fields.rebalanceHook,
			}
			gotNode, err := e.getNode(tt.args.ctx, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("getNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotNode, tt.wantNode) {
				t.Errorf("getNode() gotNode = %v, want %v", gotNode, tt.wantNode)
			}
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
				nodeMateMap:     tt.fields.nodeMateMap,
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
				nodeMateMap:     tt.fields.nodeMateMap,
				rebalanceHook:   tt.fields.rebalanceHook,
			}
			e.setNodeStatus(tt.args.ctx, tt.args.status)
		})
	}
}

func TestGetManger(t *testing.T) {
	tests := []struct {
		name string
		want *EtcdManager
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetManger(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetManger() = %v, want %v", got, tt.want)
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
