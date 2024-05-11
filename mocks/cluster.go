package mocks

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Cyprinus12138/vectory/internal/cluster"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/pkg"
	etcd "go.etcd.io/etcd/client/v3"
	"net"
)

func InitMockCluster(ctx context.Context, etcdCli *etcd.Client, num int) {
	nodeId := "mock-node-self"
	conf := &config.ClusterMetaConfig{
		ClusterName: "cluster_unittest",
		ClusterMode: config.ClusterSetting{
			Enabled:     true,
			GracePeriod: 15,
			Ttl:         5,
			LBMode:      "none",
		},
	}
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	cluster.InitEtcdManager(ctx, etcdCli, conf, nodeId)
	cluster.GetManger().Register(lis)
	cluster.GetManger().SyncCluster()

	for i := 0; i < num; i++ {
		meta := &cluster.NodeMeta{
			NodeId: fmt.Sprintf("mock-node-%d", i),
			Addr:   "127.0.0.1:0",
			Status: pkg.Healthy,
		}
		str, _ := json.Marshal(meta)
		_, err := etcdCli.Put(ctx, config.GetNodeMetaPath(meta.NodeId), string(str))
		if err != nil {
			panic(err)
		}
	}
}
