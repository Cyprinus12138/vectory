package grpc_handler

import (
	"context"
	"github.com/Cyprinus12138/vectory/internal/utils"
	"github.com/Cyprinus12138/vectory/internal/utils/func_builder"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	"github.com/Cyprinus12138/vectory/internal/utils/monitor"
	// To be updated based on proto definition.
	server "github.com/Cyprinus12138/vectory/proto/mq.example"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRpcServer struct {
	// To be updated based on proto definition.
	server.ExampleServer
	desc grpc.ServiceDesc
}

func NewServerImpl() *GRpcServer {
	return &GRpcServer{
		// To be updated based on proto definition.
		desc: server.Example_ServiceDesc,
	}
}

func SetupService(gRpcService *grpc.Server) {

	serviceImpl := NewServerImpl()
	// To be updated based on proto definition.
	server.RegisterExampleServer(gRpcService, serviceImpl)
	logger.Info(
		"register rpc service impl",
		logger.String("service", utils.ListProtoMethods(serviceImpl.desc)),
	)
}

func (s *GRpcServer) SayHello(ctx context.Context, req *server.HelloRequest) (resp *server.HelloReply, err error) {
	defer func_builder.BuildReportFunc(monitor.RpcReq)(ctx, "SayHello", "handler", &err)()

	return nil, status.Errorf(codes.Unimplemented, "method SayHello not implemented")
}
