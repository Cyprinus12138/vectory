package grpc_handler

import (
	"github.com/Cyprinus12138/vectory/internal/utils"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	pb "github.com/Cyprinus12138/vectory/proto/gen/go"
	"google.golang.org/grpc"
)

func SetupService(gRpcServer *grpc.Server) {
	coreService := NewCoreService()
	// To be updated based on proto definition.
	pb.RegisterCoreServer(gRpcServer, coreService)
	logger.Info(
		"register rpc service impl",
		logger.String("service", utils.ListProtoMethods(coreService.desc)),
	)
}

func SetupClusterService(gRpcServer *grpc.Server) {
	clusterService := NewClusterService()
	// To be updated based on proto definition.
	pb.RegisterClusterServer(gRpcServer, clusterService)
	logger.Info(
		"register cluster rpc service impl",
		logger.String("service", utils.ListProtoMethods(clusterService.desc)),
	)
}
