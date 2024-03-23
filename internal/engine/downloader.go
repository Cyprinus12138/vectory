package engine

import (
	"context"
	"path"
	"strconv"
	"strings"
)

type SourceType string

const (
	LocalPath SourceType = "local_path"
	S3        SourceType = "s3"
	Hdfs      SourceType = "hdfs"
	Ftp       SourceType = "ftp"
	Sftp      SourceType = "sftp"
)
const (
	revisionFile         = "_revision"
	indexNamePlaceHolder = "{index_name}"
	shardIdPlaceHolder   = "{shard_id}"
)

type IndexSource struct {
	Type     SourceType
	Location string

	NameFmt string
}

func (i *IndexSource) GetLatestRevisionPath() string {
	return path.Join(i.Location, revisionFile)
}

func (i *IndexSource) GetRevisionDirPath(revision int64) string {
	return path.Join(i.Location, strconv.Itoa(int(revision)))
}

func (i *IndexSource) GetShardFilePath(revision int64, shard Shard) string {
	fileName := strings.Replace(i.NameFmt, indexNamePlaceHolder, shard.IndexName, -1)
	fileName = strings.Replace(fileName, shardIdPlaceHolder, strconv.Itoa(int(shard.ShardId)), -1)
	return path.Join(i.GetRevisionDirPath(revision), fileName)
}

type Downloader interface {
	Download() (localPath string, revision int64, err error)
	DownloadUpdate(currentRevision int64) (localPath string, revision int64, err error)
}

func NewDownLoader(ctx context.Context, source *IndexSource, shard Shard) (Downloader, error) {
	switch source.Type {
	case LocalPath:
		return newLocalDl(ctx, source, shard), nil
	}

	return nil, nil
}
