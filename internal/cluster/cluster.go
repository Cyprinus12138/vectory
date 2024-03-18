package cluster

import (
	"context"
	"encoding/json"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	"github.com/Cyprinus12138/vectory/pkg"
	etcd "go.etcd.io/etcd/client/v3"
	"net"
	"time"

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
}

type clusterMeta struct {
	Addr   string         `json:"addr"`
	Status pkg.NodeStatus `json:"status"`
	CPU    float64        `json:"CPU"`
}

type clusterLoad struct {
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
		logger.String("serviceName", e.conf.ClusterName),
		logger.String("instanceId", e.nodeId),
	)

	meta := clusterMeta{
		Addr:   lis.Addr().String(),
		Status: pkg.Start,
		CPU:    -1,
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

func (e *EtcdManager) AttachLoad() {
	if e.attachedLoad {
		return
	}
	key := config.GetNodeLoadPath(e.nodeId)
	e.attachedLoad = true

	go func(key string) {
		for {
			meta := &clusterMeta{}

			usage, err := cpu.Percent(time.Duration(5)*time.Second, false) // There's a sleep logic inside the Percent method.
			if err != nil || len(usage) <= 0 {
				logger.Error("attach load get CPU usage failed", logger.Err(err))
				continue
			}
			ctx, cancel := context.WithTimeout(e.ctx, defaultRegisterTimeoutMs*time.Millisecond)
			response, err := e.etcd.Get(ctx, key, etcd.WithLimit(1))
			if err != nil {
				logger.Error("attach load get registered node meta failed", logger.Err(err))
				continue
			}
			if len(response.Kvs) <= 0 {
				logger.Error("attach load get registered node meta not found", logger.Err(err))
				continue
			}
			err = json.Unmarshal(response.Kvs[0].Value, meta)
			if err != nil {
				logger.Error("attach load decode registered node meta not found", logger.Err(err))
				continue
			}

			meta.CPU = usage[0]
			metaStr, _ := json.Marshal(meta)

			txn := e.etcd.Txn(ctx)
			_, err = txn.If(
				etcd.Compare(etcd.ModRevision(key), "=", response.Kvs[0].ModRevision),
			).Then(
				etcd.OpPut(key, string(metaStr), etcd.WithLease(e.keepaliveLease.ID)),
			).Commit()
			cancel()
			if err != nil {
				logger.Error("attach load put registered node meta failed", logger.Err(err))
				continue
			}
		}
	}(key)
}

func (e *EtcdManager) SetNodeStatus(status pkg.NodeStatus) {
	key := config.GetNodeMetaPath(e.nodeId)
	meta := &clusterMeta{}

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

func (e *EtcdManager) SyncCluster() {

}
