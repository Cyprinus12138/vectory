package processor

import (
	"context"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

type Processor interface {
	GinHandleFunc() gin.HandlerFunc
	Handle(ctx context.Context, req proto.Message) (resp proto.Message, err error)
}
