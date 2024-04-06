package grpc_client

import (
	"context"
	"fmt"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/engine"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	pb "github.com/Cyprinus12138/vectory/proto/gen/go"
	"google.golang.org/grpc"
	wrr "google.golang.org/grpc/balancer/weightedroundrobin"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"sync"
)

var (
	shardConnMap sync.Map
)

func init() {
	shardConnMap = sync.Map{} // shardConnMap key: uniqueShardKey value: *grpc.ClientConn
}

func GetShardClient(ctx context.Context, shard engine.Shard) (cli pb.ClusterClient, err error) {
	shardKey := shard.UniqueShardKey()
	var conn *grpc.ClientConn
	connRaw, ok := shardConnMap.Load(shardKey)
	if ok {
		conn, ok = connRaw.(*grpc.ClientConn)
	}

	if !ok || conn.GetState() == connectivity.Shutdown {
		url := fmt.Sprintf(config.FmtUrlForResolver, ShardResolverScheme, shardKey)
		builder := NewShardResolverBuilder()
		conn, err = grpc.Dial(
			url,
			grpc.WithResolvers(builder),
			grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"`+wrr.Name+`"}`),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			logger.CtxError(
				ctx,
				"connect to node by shard failed",
				logger.String("shardKey", shardKey),
				logger.String("url", url),
				logger.Err(err),
			)
			return nil, err
		}
		shardConnMap.Store(shardKey, conn)
	}

	return pb.NewClusterClient(conn), nil
}
