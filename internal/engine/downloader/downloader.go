package downloader

import (
	"context"
	"github.com/Cyprinus12138/vectory/internal/engine"
)

type IndexSource struct {
	Type     string
	Location string
}

type Downloader interface {
	Download() (localPath string, revision int64, err error)
	DownloadUpdate(currentRevision int64) (localPath string, revision int64, err error)
}

func NewDownLoader(ctx context.Context, source *IndexSource, shard engine.Shard) (Downloader, error) {
	return nil, nil
}
