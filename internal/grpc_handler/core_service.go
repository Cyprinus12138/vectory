package grpc_handler

import (
	"context"
	"github.com/Cyprinus12138/vectory/internal/processor"
	pb "github.com/Cyprinus12138/vectory/proto/gen/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CoreService struct {
	pb.CoreServer
	desc grpc.ServiceDesc
}

func NewCoreService() *CoreService {
	return &CoreService{
		// To be updated based on proto definition.
		desc: pb.Core_ServiceDesc,
	}
}

func (s *CoreService) Search(ctx context.Context, request *pb.SearchRequest) (*pb.SearchResponse, error) {
	proc := processor.SearchProcessor{}
	return proc.Handle(ctx, request)
}

func (s *CoreService) ListIndex(ctx context.Context, request *pb.ListIndexRequest) (*pb.ListIndexResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ListIndex not implemented")
}

func (s *CoreService) CreateIndex(ctx context.Context, request *pb.CreateIndexRequest) (*pb.CreateIndexResponse, error) {
	return nil, status.Error(codes.Unimplemented, "CreateIndex not implemented")
}

func (s *CoreService) GetIndexStat(ctx context.Context, request *pb.GetIndexStatRequest) (*pb.GetIndexStatResponse, error) {
	return nil, status.Error(codes.Unimplemented, "GetIndexStat not implemented")
}

func (s *CoreService) GetIndexMeta(ctx context.Context, request *pb.GetIndexMetaRequest) (*pb.GetIndexMetaResponse, error) {
	return nil, status.Error(codes.Unimplemented, "GetIndexMeta not implemented")
}
