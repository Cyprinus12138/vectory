package grpc_client

import (
	"fmt"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	etcd "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	gr "google.golang.org/grpc/resolver"
)

var (
// Add your grpc client here
)

func Init(e *etcd.Client) error {
	/* Init resolver here.
	etcdResolver, err := resolver.NewBuilder(e)
	if err != nil {
		return err
	}
	*/
	/* Call your client init method here
	err = initAccount(etcdResolver)
	if err != nil {
		return err
	}
	*/
	return nil
}

// connectToServer Should be called inside client init method.
func connectToServer(r gr.Builder, svcName string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(
		fmt.Sprintf(config.FmtEtcdSvcResolveFmt, svcName),
		grpc.WithResolvers(r),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	logger.Info(
		"connect to server",
		logger.String("service", svcName),
		logger.String("target", conn.Target()),
		logger.String("state", conn.GetState().String()),
	)
	return conn, nil
}

/* Add your init method and getter here
func initAccount(r gr.Builder) (err error) {
	conn, err := connectToServer(r, ap.ClusterName)
	if err != nil {
		return err
	}
	logger.Info(
		"connect to server",
		logger.String("service", ap.ClusterName),
		logger.String("target", conn.Target()),
		logger.String("state", conn.GetState().String()),
	)
	accountClient = ac.NewAccountClient(conn)
	return nil
}


func GetAccountClient(ctx context.Context) ac.AccountClient {
	if accountClient == nil {
		logger.CtxWarn(ctx, "client not init", logger.String("svc", ap.ClusterName))
	}
	return accountClient
}

*/
