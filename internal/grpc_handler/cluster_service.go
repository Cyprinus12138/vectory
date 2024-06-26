package grpc_handler

import (
	"context"
	"github.com/Cyprinus12138/vectory/internal/processor"
	pb "github.com/Cyprinus12138/vectory/proto/gen/go"
	"google.golang.org/grpc"
)

type ClusterService struct {
	pb.ClusterServer
	desc grpc.ServiceDesc
}

func NewClusterService() *ClusterService {
	return &ClusterService{
		// To be updated based on proto definition.
		desc: pb.Cluster_ServiceDesc,
	}
}

func (s *ClusterService) SearchShard(ctx context.Context, request *pb.SearchShardRequest) (*pb.SearchShardResponse, error) {
	proc := processor.SearchShardProcessor{}
	return proc.Handle(ctx, request)
}
