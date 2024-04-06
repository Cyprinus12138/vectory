package processor

import (
	"context"
	"fmt"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/engine"
	"github.com/Cyprinus12138/vectory/internal/utils"
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
// TODO add log error handling and rpc searchShard
func (s *SearchProcessor) Handle(ctx context.Context, req *pb.SearchRequest) (resp *pb.SearchResponse, err error) {
	idxManager := engine.GetManager()
	resp.Results = make([]*pb.SearchResult, len(req.Input))

	for i, vector := range req.GetInput() {
		shardResults, iErr := idxManager.Search(ctx, req.GetIndexName(), vector.GetVector(), int64(req.GetLimit()))
		if iErr != nil {
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
		for j, result := range shardResults {
			var shardErr error
			shardResultsPb[j] = &pb.SearchResult{}
			labels := result.Result

			if result.ToRoute {
				// TODO Fetch remote result and replace the labels value.
			}
			shardResultsPb[j].Result = make([]*pb.Item, len(labels))
			shardResultsPb[j].Error = shardErr.Error()
			for idx, label := range labels {
				shardResultsPb[j].Result[idx] = &pb.Item{
					Id:    strconv.FormatInt(label.Label, 10),
					Score: label.Distance,
					Doc:   "", // TODO coming soon!
				}
			}
			itemResultPb[j] = shardResultsPb[j].GetResult()
		}
		if len(shardResults) > 1 {
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
				mergedErr = fmt.Sprintf("%s|%s", mergedErr, searchResult.GetError())
			}
			if errorFlag {
				result.Error = mergedErr
			}
		} else {
			resp.Results[i] = shardResultsPb[0]
		}
	}

	return resp, err
}
