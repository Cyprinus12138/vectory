package grpc_handler

import (
	"context"
	"github.com/Cyprinus12138/vectory/internal/processor"
	pb "github.com/Cyprinus12138/vectory/proto/gen/go"
	"google.golang.org/grpc"
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
	//TODO implement me
	panic("implement me")
}

func (s *CoreService) CreateIndex(ctx context.Context, request *pb.CreateIndexRequest) (*pb.CreateIndexResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s *CoreService) GetIndexStat(ctx context.Context, request *pb.GetIndexStatRequest) (*pb.GetIndexStatResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s *CoreService) GetIndexMeta(ctx context.Context, request *pb.GetIndexMetaRequest) (*pb.GetIndexMetaResponse, error) {
	//TODO implement me
	panic("implement me")
}
