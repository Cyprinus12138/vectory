package engine

import (
	"context"
	"fmt"
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
	Type     SourceType `json:"type,omitempty" yaml:"type"`
	Location string     `json:"location,omitempty" yaml:"location"`
	NameFmt  string     `json:"name_fmt,omitempty" yaml:"name_fmt"`

	// S3-specific fields.
	Endpoint  string `json:"endpoint,omitempty" yaml:"endpoint"` // e.g. "play.min.io:9000", "s3.amazonaws.com"
	AccessKey string `json:"access_key,omitempty" yaml:"access_key"`
	SecretKey string `json:"secret_key,omitempty" yaml:"secret_key"`
	UseSSL    bool   `json:"use_ssl,omitempty" yaml:"use_ssl"`
}

func (i *IndexSource) GetLatestRevisionPath() string {
	return path.Join(i.Location, revisionFile)
}

func (i *IndexSource) GetRevisionDirPath(revision int64) string {
	return path.Join(i.Location, strconv.Itoa(int(revision)))
}

// GetShardRelPath returns the revision-relative shard file path, e.g. "42/idx_shard_0.faiss".
func (i *IndexSource) GetShardRelPath(revision int64, shard Shard) string {
	fileName := strings.Replace(i.NameFmt, indexNamePlaceHolder, shard.IndexName, -1)
	fileName = strings.Replace(fileName, shardIdPlaceHolder, strconv.Itoa(int(shard.ShardId)), -1)
	return path.Join(strconv.Itoa(int(revision)), fileName)
}

func (i *IndexSource) GetShardFilePath(revision int64, shard Shard) string {
	return path.Join(i.Location, i.GetShardRelPath(revision, shard))
}

type Downloader interface {
	Download() (localPath string, revision int64, err error)
	DownloadUpdate(currentRevision int64) (localPath string, revision int64, err error)
}

func NewDownLoader(ctx context.Context, source *IndexSource, shard Shard) (Downloader, error) {
	switch source.Type {
	case LocalPath:
		return newLocalDl(ctx, source, shard), nil
	case S3:
		return newS3Dl(ctx, source, shard)
	}
	return nil, fmt.Errorf("unsupported source type: %s", source.Type)
}
