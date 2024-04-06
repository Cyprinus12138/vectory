package grpc_client

import (
	"github.com/Cyprinus12138/vectory/internal/cluster"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/engine"
	"github.com/robfig/cron/v3"
	robin "google.golang.org/grpc/balancer/weightedroundrobin"
	"google.golang.org/grpc/resolver"
	"sync"
)

const (
	// Define a unique scheme for your resolver.
	ShardResolverScheme = "vec-idx"

	// Define a fixed address for demonstration purposes.
	// In a real-world scenario, you would dynamically determine the address.
	resolveInterval = "@every 30s"
)

var scheduler = cron.New()

func GetScheduler() *cron.Cron {
	return scheduler
}

func init() {
	GetScheduler().Start()
}

// ShardResolverBuilder implements the resolver.Builder interface.
type ShardResolverBuilder struct{}

func NewShardResolverBuilder() *ShardResolverBuilder {
	return &ShardResolverBuilder{}
}

func (*ShardResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &vectoryResolver{
		target: target,
		cc:     cc,
		wg:     sync.WaitGroup{},
	}
	r.start()
	return r, nil
}

func (*ShardResolverBuilder) Scheme() string {
	return ShardResolverScheme
}

// vectoryResolver implements the resolver.Resolver interface.
type vectoryResolver struct {
	target   resolver.Target
	cc       resolver.ClientConn
	routings []*cluster.Routing
	entry    cron.EntryID
	wg       sync.WaitGroup
}

func (r *vectoryResolver) start() {
	engineManager := engine.GetManager()
	engineManager.RegisterListener(r.target.Endpoint(), r)
	r.Update()
	r.entry, _ = GetScheduler().AddFunc(resolveInterval, func() {
		r.ResolveNow(resolver.ResolveNowOptions{})
	})

}

func (r *vectoryResolver) ResolveNow(o resolver.ResolveNowOptions) {
	r.wg.Add(1)
	defer r.wg.Done()
	var err error

	tgtUniqueShard := &engine.Shard{}
	tgtUniqueShard.FromString(r.target.Endpoint())

	if !tgtUniqueShard.Valid() {
		r.cc.ReportError(config.ErrResolveIndexShard)
		return
	}
	engineManager := engine.GetManager()
	r.routings, err = engineManager.ResolveUniqueShard(tgtUniqueShard)
	if err != nil {
		r.cc.ReportError(err)
		return
	}
	addrs := convertRoutingToAddrs(r.routings)
	if len(addrs) == 0 {
		r.cc.ReportError(config.ErrResolveNoAvailableNode)
		return
	}
	r.cc.UpdateState(resolver.State{Addresses: addrs})
}

func (r *vectoryResolver) Close() {
	engine.GetManager().UnregisterListener(r.target.Endpoint())
	GetScheduler().Remove(r.entry)
	r.wg.Wait()
}

func (r *vectoryResolver) Update() {
	go r.ResolveNow(resolver.ResolveNowOptions{})
}

func convertRoutingToAddrs(routings []*cluster.Routing) []resolver.Address {
	res := make([]resolver.Address, 0, len(routings))

	for _, routing := range routings {
		if routing.Weight() == 0 {
			continue
		}
		addr := resolver.Address{
			Addr:       routing.Node.Addr,
			ServerName: routing.Node.NodeId,
		}
		addr = robin.SetAddrInfo(
			addr,
			robin.AddrInfo{
				Weight: routing.Weight(),
			},
		)
		res = append(res, addr)
	}
	return res
}
