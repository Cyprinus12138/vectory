package engine

import (
	"context"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/utils"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	"os"
	"strconv"
)

type localDl struct {
	ctx    context.Context
	source *IndexSource

	shard Shard
}

func (l *localDl) Download() (localPath string, revision int64, err error) {
	log := logger.DefaultLoggerWithCtx(l.ctx).With(logger.Interface("source", l.source), logger.String(
		"shardKey",
		l.shard.ShardKey(),
	))
	revision, err = l.readRevision()
	if err != nil {
		log.Error(
			"get revision failed",
			logger.String("revisionFile", l.source.GetLatestRevisionPath()),
			logger.Err(err),
		)
		return "", 0, config.ErrGetRevisionFailed
	}

	filePath := l.source.GetShardFilePath(revision, l.shard)
	if !utils.FileExists(filePath) {
		log.Error("index file not found", logger.String("path", filePath))
		return "", 0, config.ErrShardFileNotFound
	}
	return filePath, revision, nil
}

func (l *localDl) DownloadUpdate(currentRevision int64) (localPath string, revision int64, err error) {
	log := logger.DefaultLoggerWithCtx(l.ctx).With(logger.Interface("source", l.source), logger.String(
		"shardKey",
		l.shard.ShardKey(),
	))

	revision, err = l.readRevision()
	if err != nil {
		log.Error(
			"get revision failed",
			logger.String("revisionFile", l.source.GetLatestRevisionPath()),
			logger.Err(err),
		)
		return "", 0, config.ErrGetRevisionFailed
	}

	if revision <= currentRevision {
		log.Info(
			"shard already up-to-date",
			logger.Int64("currentRevision", currentRevision),
			logger.Int64("revision", revision),
		)
		return "", revision, config.ErrIndexRevisionUpToDate
	}
	return l.Download()
}

func (l *localDl) readRevision() (int64, error) {
	log := logger.DefaultLoggerWithCtx(l.ctx).With(logger.Interface("source", l.source), logger.String(
		"shardKey",
		l.shard.ShardKey(),
	))
	bytes, err := os.ReadFile(l.source.GetLatestRevisionPath())
	if err != nil {
		log.Error(
			"read revision file failed",
			logger.String("revisionFile", l.source.GetLatestRevisionPath()),
			logger.Err(err),
		)
		return 0, config.ErrGetRevisionFailed
	}

	revisionInt, err := strconv.Atoi(string(bytes))
	if err != nil {
		log.Error(
			"read revision file failed",
			logger.String("revisionFile", l.source.GetLatestRevisionPath()),
			logger.String("content", string(bytes)),
			logger.Err(err),
		)
		return 0, config.ErrGetRevisionFailed
	}
	revision := int64(revisionInt)
	return revision, nil
}

func newLocalDl(ctx context.Context, source *IndexSource, shard Shard) *localDl {
	return &localDl{
		ctx:    ctx,
		source: source,
		shard:  shard,
	}
}
