package cluster

import (
	"context"
	"encoding/json"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	"github.com/Cyprinus12138/vectory/pkg"
	"go.etcd.io/etcd/api/v3/mvccpb"
	etcd "go.etcd.io/etcd/client/v3"
	"net"
	"time"

	"github.com/serialx/hashring"
	"github.com/shirou/gopsutil/cpu"
)

const defaultRegisterTimeoutMs = 5000

var manager *EtcdManager

type EtcdManager struct {
	ctx            context.Context
	etcd           *etcd.Client
	conf           *config.ClusterMetaConfig
	nodeId         string
	keepaliveLease *etcd.LeaseGrantResponse

	attachedLoad bool
	// TODO Add mutex to protect the ring.
	clusterHashRing *hashring.HashRing
}

type nodeMeta struct {
	NodeId string         `json:"node_id"`
	Addr   string         `json:"addr"`
	Status pkg.NodeStatus `json:"status"`
}

type nodeLoad struct {
	CPU float64 `json:"CPU"`
}

func InitEtcdManager(ctx context.Context, e *etcd.Client, conf *config.ClusterMetaConfig, nodeId string) {
	manager = &EtcdManager{
		ctx:    ctx,
		etcd:   e,
		conf:   conf,
		nodeId: nodeId,
	}
}

func GetManger() *EtcdManager {
	return manager
}

func (e *EtcdManager) Register(lis net.Listener) (err error) {
	conf := e.conf.ClusterMode
	logger.Info(
		"registering the rpc service",
		logger.String("clusterName", e.conf.ClusterName),
		logger.String("instanceId", e.nodeId),
	)

	meta := nodeMeta{
		NodeId: e.nodeId,
		Addr:   lis.Addr().String(),
		Status: pkg.Start,
	}
	metaStr, _ := json.Marshal(meta)

	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(e.ctx, defaultRegisterTimeoutMs*time.Millisecond)
	defer cancel()
	e.keepaliveLease, err = e.etcd.Grant(ctx, conf.Ttl)
	if err != nil {
		return err
	}

	var keepAliveChan <-chan *etcd.LeaseKeepAliveResponse
	keepAliveChan, err = e.etcd.Lease.KeepAlive(e.ctx, e.keepaliveLease.ID) // Remember to use e.ctx, because the ctx with timeout will be canceled after the register method finished.
	if err != nil {
		return err
	}
	go func() {
		for {
			<-keepAliveChan
		}
	}()

	_, err = e.etcd.Put(ctx, config.GetNodeMetaPath(e.nodeId), string(metaStr), etcd.WithLease(e.keepaliveLease.ID))

	if err != nil {
		return err
	}
	return nil
}

func (e *EtcdManager) Unregister() {
	logger.Info(
		"unregistering the node",
		logger.String("clusterName", e.conf.ClusterName),
		logger.String("instanceId", e.nodeId),
	)
	_, err := e.etcd.Delete(e.ctx, config.GetNodeMetaPath(e.nodeId))
	if err != nil {
		logger.Error(
			"unregistering the node failed",
			logger.String("clusterName", e.conf.ClusterName),
			logger.String("instanceId", e.nodeId),
			logger.Err(err),
		)
	}
}

func (e *EtcdManager) ReportLoad() {
	if e.attachedLoad {
		return
	}
	key := config.GetNodeLoadPath(e.nodeId)
	e.attachedLoad = true

	go func(key string) {
		for {
			load := &nodeLoad{}

			usage, err := cpu.Percent(time.Duration(5)*time.Second, false) // There's a sleep logic inside the Percent method.
			if err != nil || len(usage) <= 0 {
				logger.Error("attach load get CPU usage failed", logger.Err(err))
				continue
			}
			ctx, cancel := context.WithTimeout(e.ctx, defaultRegisterTimeoutMs*time.Millisecond)
			response, err := e.etcd.Get(ctx, key, etcd.WithLimit(1))
			if err != nil {
				logger.Error("attach load get registered node load failed", logger.Err(err))
				continue
			}
			if len(response.Kvs) <= 0 {
				logger.Error("attach load get registered node load not found", logger.Err(err))
				continue
			}
			err = json.Unmarshal(response.Kvs[0].Value, load)
			if err != nil {
				logger.Error("attach load decode registered node load not found", logger.Err(err))
				continue
			}

			load.CPU = usage[0]
			metaStr, _ := json.Marshal(load)

			txn := e.etcd.Txn(ctx)
			_, err = txn.If(
				etcd.Compare(etcd.ModRevision(key), "=", response.Kvs[0].ModRevision),
			).Then(
				etcd.OpPut(key, string(metaStr), etcd.WithLease(e.keepaliveLease.ID)),
			).Commit()
			cancel()
			if err != nil {
				logger.Error("attach load put registered node load failed", logger.Err(err))
				continue
			}
		}
	}(key)
}

func (e *EtcdManager) SetNodeStatus(status pkg.NodeStatus) {
	key := config.GetNodeMetaPath(e.nodeId)
	meta := &nodeMeta{}

	response, err := e.etcd.Get(e.ctx, key, etcd.WithLimit(1))
	if err != nil {
		logger.Error("attach load get registered node meta failed", logger.Err(err))
		return
	}
	if len(response.Kvs) <= 0 {
		logger.Error("attach load get registered node meta not found", logger.Err(err))
		return
	}

	err = json.Unmarshal(response.Kvs[0].Value, meta)
	if err != nil {
		logger.Error("attach load decode registered node meta not found", logger.Err(err))
		return
	}

	meta.Status = status
	metaStr, _ := json.Marshal(meta)
	ctx, cancel := context.WithTimeout(e.ctx, defaultRegisterTimeoutMs*time.Millisecond)
	defer cancel()
	txn := e.etcd.Txn(ctx)
	_, err = txn.If(
		etcd.Compare(etcd.ModRevision(key), "=", response.Kvs[0].ModRevision),
	).Then(
		etcd.OpPut(key, string(metaStr), etcd.WithLease(e.keepaliveLease.ID)),
	).Commit()
	if err != nil {
		logger.Error("attach load put registered node meta failed", logger.Err(err))
		return
	}
}

func (e *EtcdManager) SyncCluster() error {
	ctx, cancel := context.WithTimeout(e.ctx, defaultRegisterTimeoutMs*time.Millisecond)
	defer cancel()
	resp, err := e.etcd.Get(ctx, config.GetNodeMetaPath(config.GetNodeMetaPathPrefix()), etcd.WithPrefix())
	if err != nil {
		logger.Error("get node meta failed", logger.Err(err))
		return err
	}
	nodes := make([]string, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		meta := &nodeMeta{}
		jErr := json.Unmarshal(kv.Value, meta)
		if jErr != nil {
			logger.Error("invalid node meta", logger.String("meta", string(kv.Value)), logger.Err(jErr))
			return jErr
		}
		if meta.Status == pkg.Inactive {
			continue
		}
		nodes = append(nodes, meta.NodeId)
	}

	e.clusterHashRing = hashring.New(nodes)

	go func() {
		watch := e.etcd.Watch(e.ctx, config.GetNodeMetaPathPrefix(), etcd.WithPrefix())
		var watchResp etcd.WatchResponse
		select {
		case watchResp = <-watch:
			if watchResp.Canceled {
				logger.Fatal("watch cluster canceled",
					logger.String("clusterName", e.conf.ClusterName),
					logger.String("instanceId", e.nodeId),
					logger.Err(err),
				)
				return
			}
			for _, event := range watchResp.Events {
				meta := &nodeMeta{}
				jErr := json.Unmarshal(event.Kv.Value, meta)
				if jErr != nil {
					logger.Error("invalid node meta", logger.String("meta", string(event.Kv.Value)), logger.Err(jErr))
				}
				if event.IsCreate() {
					e.clusterHashRing.AddNode(meta.NodeId)
				}
				if event.Type == mvccpb.DELETE {
					e.clusterHashRing.RemoveNode(meta.NodeId)
				}
				if event.IsModify() && meta.Status == pkg.Inactive {
					e.clusterHashRing.RemoveNode(meta.NodeId)
				}
			}
		}
	}()
	return nil
}

func (e *EtcdManager) NeedLoad(key string) bool {
	node, ok := e.clusterHashRing.GetNode(key)
	return ok && node == e.nodeId
}
