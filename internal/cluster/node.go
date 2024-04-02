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
	"sync"
	"time"

	"github.com/serialx/hashring"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

const (
	defaultRegisterTimeoutMs = 5000
	initialWeight            = 1000
)

var manager *EtcdManager

type EtcdManager struct {
	ctx            context.Context
	etcd           *etcd.Client
	conf           *config.ClusterMetaConfig
	nodeId         string
	keepaliveLease *etcd.LeaseGrantResponse

	// attachedLoad indicates whether already reporting the loads.
	attachedLoad    bool
	clusterHashRing *hashring.HashRing
	nodeMateMap     sync.Map // Key: nodeId Value: NodeMeta
	rebalanceHook   func(ctx context.Context) error
}

type NodeMeta struct {
	NodeId string         `json:"node_id"`
	Addr   string         `json:"addr"`
	Status pkg.NodeStatus `json:"status"`
}

type NodeLoad struct {
	CPU     float64 `json:"cpu"`
	MemFree uint64  `json:"mem_free"`
}

type Routing struct {
	Node *NodeMeta
	Load *NodeLoad
}

func (r *Routing) Weight() uint32 {
	if r.Node == nil || r.Node.Addr == "" {
		return 0
	}

	var weight uint32 = initialWeight
	if r.Load != nil && r.Load.CPU != 0 {
		weight = uint32((1 - r.Load.CPU) * float64(weight))
	}
	switch r.Node.Status {
	case pkg.Healthy:
		return weight
	case pkg.Rebalancing:
	case pkg.Unhealthy:
		return uint32(float32(weight) * 0.1)
	case pkg.Init:
	case pkg.Inactive:
	case pkg.Start:
		return 0
	}
	return 0
}

func InitEtcdManager(ctx context.Context, e *etcd.Client, conf *config.ClusterMetaConfig, nodeId string) {
	manager = &EtcdManager{
		ctx:         ctx,
		etcd:        e,
		conf:        conf,
		nodeId:      nodeId,
		nodeMateMap: sync.Map{},
	}
}

func GetManger() *EtcdManager {
	return manager
}

// Register will register the node to EtCD,
// we utilise the keepalive lease to maintain the node online in the cluster
func (e *EtcdManager) Register(lis net.Listener) (err error) {
	conf := e.conf.ClusterMode
	logger.Info(
		"registering the rpc service",
		logger.String("clusterName", e.conf.ClusterName),
		logger.String("instanceId", e.nodeId),
	)

	meta := NodeMeta{
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

	// The status updating information is consumed here, updating the status info on the EtCD.
	go func(ctx context.Context) {
		select {
		case status := <-pkg.StatusUpdating:
			e.setNodeStatus(ctx, status)
		}
	}(e.ctx)

	return nil
}

// Unregister is not really necessary actually because the node meta will be removed by the keepalive lease mechanism.
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

// ReportLoad will update the load information of the node to the EtCD regularly.
func (e *EtcdManager) ReportLoad(policy config.LBModeType) {
	if e.attachedLoad {
		return
	}
	var (
		reportCPU, reportMEM bool
	)
	switch policy.Standard() {
	case config.LBCPU:
		reportCPU = true
		reportMEM = true
	case config.LBNone:
	default:
		return
	}
	key := config.GetNodeLoadPath(e.nodeId)
	e.attachedLoad = true

	go func(key string) {
		for {
			load := &NodeLoad{}
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
			if reportCPU {
				cpuUsage, err := cpu.Percent(time.Duration(5)*time.Second, false) // There's a sleep logic inside the Percent method.
				if err != nil || len(cpuUsage) <= 0 {
					logger.Error("attach load get CPU usage failed", logger.Err(err))
					continue
				}
				load.CPU = cpuUsage[0]
			}
			if reportMEM {
				memStat, err := mem.VirtualMemory()
				if err != nil {
					logger.Error("attach load get MEM free failed", logger.Err(err))
					continue
				}
				load.MemFree = memStat.Available
			}

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

func (e *EtcdManager) setNodeStatus(ctx context.Context, status pkg.NodeStatus) {
	key := config.GetNodeMetaPath(e.nodeId)

	meta, err := e.getNodeMeta(ctx, e.nodeId)
	if err != nil {
		logger.CtxError(ctx, "can not find meta for myself")
		return
	}

	meta.Status = status
	metaStr, _ := json.Marshal(meta)
	ctx, cancel := context.WithTimeout(ctx, defaultRegisterTimeoutMs*time.Millisecond)
	defer cancel()
	_, err = e.etcd.Put(ctx, key, string(metaStr), etcd.WithLease(e.keepaliveLease.ID))
	if err != nil {
		logger.Error("put node meta when set node status failed", logger.Err(err))
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
		meta := &NodeMeta{}
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
					logger.Err(watchResp.Err()),
				)
				return
			}
			for _, event := range watchResp.Events {
				meta := &NodeMeta{}
				jErr := json.Unmarshal(event.Kv.Value, meta)
				if jErr != nil {
					logger.Error("invalid node meta", logger.String("meta", string(event.Kv.Value)), logger.Err(jErr))
				}
				if event.IsCreate() {
					e.clusterHashRing.AddNode(meta.NodeId)
					e.nodeMateMap.Store(meta.NodeId, meta)
					err = e.rebalanceHook(e.ctx)
					if err != nil {
						logger.Error("rebalance failed", logger.String("meta", string(event.Kv.Value)), logger.Err(err))
					}
				}
				if event.Type == mvccpb.DELETE {
					e.nodeMateMap.Delete(meta.NodeId)
					e.clusterHashRing.RemoveNode(meta.NodeId)
					err = e.rebalanceHook(e.ctx)
					if err != nil {
						logger.Error("rebalance failed", logger.String("meta", string(event.Kv.Value)), logger.Err(err))
					}
				}
				if event.IsModify() {
					e.nodeMateMap.Store(meta.NodeId, meta)
					if meta.Status == pkg.Inactive {
						e.clusterHashRing.RemoveNode(meta.NodeId)
						err = e.rebalanceHook(e.ctx)
						if err != nil {
							logger.Error("rebalance failed", logger.String("meta", string(event.Kv.Value)), logger.Err(err))
						}
					}
				}
			}
		}
	}()
	return nil
}

// NeedLoad returns whether the shard is loaded in current node.
func (e *EtcdManager) NeedLoad(key string) bool {
	node, ok := e.clusterHashRing.GetNode(key)
	return ok && node == e.nodeId
}

func (e *EtcdManager) getNodeMeta(ctx context.Context, nodeId string) (*NodeMeta, error) {
	log := logger.DefaultLoggerWithCtx(ctx).With(logger.String("node", nodeId))

	raw, ok := e.nodeMateMap.Load(nodeId)
	if !ok {
		log.Error("node is no long valid")
		return nil, config.ErrNodeNotAvailable
	}
	node, ok := raw.(*NodeMeta)
	if !ok {
		log.Error("type conversion failed", logger.Interface("NodeMeta", raw))
		return nil, config.ErrTypeAssertion
	}

	return node, nil
}

func (e *EtcdManager) getNode(ctx context.Context, key string) (node *NodeMeta, err error) {
	log := logger.DefaultLoggerWithCtx(ctx).With(logger.String("key", key))
	nodeId, ok := e.clusterHashRing.GetNode(key)
	if !ok {
		log.Error("no available node in the cluster")
		return nil, config.ErrNodeNotAvailable
	}

	log = log.With(logger.String("node", nodeId))
	node, err = e.getNodeMeta(ctx, nodeId)
	if err != nil {
		log.Error("get node meta failed", logger.Err(err))
	}

	switch node.Status {
	case pkg.Init:
	case pkg.Start:
	case pkg.Inactive:
		log.Warn("node is not available but not healthy", logger.String("status", node.Status.ToString()))
		return nil, config.ErrNodeNotAvailable
	case pkg.Healthy:
		return node, nil
	case pkg.Rebalancing:
	case pkg.Unhealthy:
		log.Warn("node is available but not healthy", logger.String("status", node.Status.ToString()))
		return node, nil
	}
	return nil, config.ErrUnknownStatus
}

func (e *EtcdManager) getLoad(ctx context.Context, nodeId string) (load *NodeLoad, err error) {
	log := logger.DefaultLoggerWithCtx(ctx).With(logger.String("nodeId", nodeId))
	key := config.GetNodeLoadPath(e.nodeId)
	ctx, cancel := context.WithTimeout(e.ctx, defaultRegisterTimeoutMs*time.Millisecond)
	defer cancel()
	response, err := e.etcd.Get(ctx, key, etcd.WithLimit(1))
	if err != nil {
		log.Error("attach load get registered node load failed", logger.Err(err))
		return nil, err
	}
	if len(response.Kvs) <= 0 {
		log.Error("attach load get registered node load not found", logger.Err(err))
		return nil, err
	}
	err = json.Unmarshal(response.Kvs[0].Value, load)
	if err != nil {
		log.Error("attach load decode registered node load not found", logger.Err(err))
		return nil, err
	}
	return load, nil
}

func (e *EtcdManager) Route(ctx context.Context, key string) (routing *Routing, err error) {
	log := logger.DefaultLoggerWithCtx(ctx).With(logger.String("key", key))
	node, err := e.getNode(ctx, key)
	if err != nil {
		log.Error("getNode failed", logger.Err(err))
		return nil, err
	}
	routing = &Routing{
		Node: node,
	}

	if e.attachedLoad {
		routing.Load, err = e.getLoad(ctx, node.NodeId)
		if err != nil {
			log.Error("getLoad failed", logger.Err(err))
			return routing, err
		}
	}
	return routing, nil
}

func (e *EtcdManager) SetRebalanceHook(f func(ctx context.Context) error) {
	e.rebalanceHook = f
}
