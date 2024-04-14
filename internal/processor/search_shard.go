package processor

import (
	"context"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/engine"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	pb "github.com/Cyprinus12138/vectory/proto/gen/go"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"net/http"
)

type SearchShardProcessor struct {
}

func (s *SearchShardProcessor) GinHandleFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &pb.SearchShardRequest{}
		err := c.BindJSON(req)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		resp, err := s.Handle(c, req)
		if err != nil {
			if errors.Is(err, config.ErrShardNotFound) {
				c.Status(http.StatusNotFound)
			} else {
				c.Status(http.StatusInternalServerError)
			}
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// Handle is called by gRPC handlers
// TODO add log error handling and rpc searchShard
func (s *SearchShardProcessor) Handle(ctx context.Context, req *pb.SearchShardRequest) (resp *pb.SearchShardResponse, err error) {
	log := logger.DefaultLoggerWithCtx(ctx)
	idxManager := engine.GetManager()
	resp = &pb.SearchShardResponse{}
	resp.Result = &pb.SearchResult{}
	resp.Result.Result = make([]*pb.Item, 0, req.GetLimit())
	shard := &engine.Shard{}
	shard.FromString(req.GetShardKey())

	searchResult, err := idxManager.SearchShard(ctx, *shard, req.Input.GetVector(), int64(req.GetLimit()))
	if err != nil {
		log.Error("search shard failed", logger.String("shardKey", shard.UniqueShardKey()), logger.Err(err))
		resp.Result.Error = err.Error()
		return resp, err
	}

	for _, label := range searchResult.Result {
		resp.Result.Result = append(resp.Result.Result, &pb.Item{
			Id:    label.Label,
			Score: label.Distance,
			Doc:   "",
		})
	}

	return resp, err
}
