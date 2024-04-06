package processor

import (
	"context"
	"fmt"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/engine"
	"github.com/Cyprinus12138/vectory/internal/grpc_client"
	"github.com/Cyprinus12138/vectory/internal/utils"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	pb "github.com/Cyprinus12138/vectory/proto/gen/go"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
)

type SearchProcessor struct {
}

func (s *SearchProcessor) GinHandleFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &pb.SearchRequest{}
		err := c.BindJSON(req)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		resp, err := s.Handle(c, req)
		if err != nil {
			if errors.Is(err, config.ErrIndexNotFound) {
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
func (s *SearchProcessor) Handle(ctx context.Context, req *pb.SearchRequest) (resp *pb.SearchResponse, err error) {
	log := logger.DefaultLoggerWithCtx(ctx)
	idxManager := engine.GetManager()
	resp = &pb.SearchResponse{}
	resp.Results = make([]*pb.SearchResult, len(req.Input))

	for i, vector := range req.GetInput() { // Iterate inputs, batch search for multiple input.
		shardResults, iErr := idxManager.Search(ctx, req.GetIndexName(), vector.GetVector(), int64(req.GetLimit()))
		if iErr != nil {
			log.Error(
				"failed to search index locally",
				logger.String("indexName", req.GetIndexName()),
				logger.Err(err),
			)
			resp.Results[i] = &pb.SearchResult{
				Error: iErr.Error(),
			}
			continue
		}
		if len(shardResults) <= 0 {
			return nil, config.ErrIndexNotFound
		}
		shardResultsPb := make([]*pb.SearchResult, len(shardResults))
		itemResultPb := make([][]*pb.Item, len(shardResults))
		for j, result := range shardResults { // Iterate shards, search single index with shards if any.
			var shardErr error
			shardResultsPb[j] = &pb.SearchResult{}
			labels := result.Result

			if !result.ToRoute {
				shardResultsPb[j].Result = make([]*pb.Item, len(labels))
				shardResultsPb[j].Error = shardErr.Error()
				for idx, label := range labels {
					shardResultsPb[j].Result[idx] = &pb.Item{
						Id:    strconv.FormatInt(label.Label, 10),
						Score: label.Distance,
						Doc:   "", // TODO coming soon!
					}
				}
			} else {
				cli, err := grpc_client.GetShardClient(ctx, result.Shard)
				if err == nil {
					shardResp, err := cli.SearchShard(ctx, &pb.SearchShardRequest{Input: vector, Limit: req.GetLimit()})
					if err != nil {
						shardResultsPb[j].Error = err.Error()
						log.Error(
							"failed to search remote shard",
							logger.String("shardKey", result.Shard.UniqueShardKey()),
							logger.Err(err),
						)
					}
					if shardResp != nil {
						shardResultsPb[j] = shardResp.GetResult()
					}
				} else {
					shardResultsPb[j].Error = err.Error()
					log.Error(
						"failed to get client by shard",
						logger.String("shardKey", result.Shard.UniqueShardKey()),
						logger.Err(err),
					)
				}
			}
			itemResultPb[j] = shardResultsPb[j].GetResult()
		} // Shard search end.
		if len(shardResults) > 1 { // If the index consists of multiple shards, merge the results per shard.
			var (
				errorFlag bool
				mergedErr string
			)
			result := &pb.SearchResult{}
			result.Result = utils.MergeSortedLists[*pb.Item](itemResultPb, func(i, j *pb.Item) bool {
				return true
			}, int(req.GetLimit()))

			for _, searchResult := range shardResultsPb {
				if len(searchResult.GetError()) > 0 {
					errorFlag = true
				}
				// Errors per shard
				mergedErr = fmt.Sprintf("%s|%s", mergedErr, searchResult.GetError())
			}
			if errorFlag {
				result.Error = mergedErr
			}
			resp.Results[i] = result
		} else {
			resp.Results[i] = shardResultsPb[0]
		}
	} // Search multiple input end

	return resp, err
}
