package engine

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// s3Dl downloads index files from an S3-compatible bucket using minio-go.
//
// Required IndexSource fields:
//   - Location: "s3://bucket/prefix" or "bucket/prefix"
//   - Endpoint: S3/MinIO host, e.g. "play.min.io:9000" or "s3.amazonaws.com"
//   - AccessKey, SecretKey: credentials for the bucket
//   - UseSSL: whether to use HTTPS
type s3Dl struct {
	ctx    context.Context
	source *IndexSource
	shard  Shard

	client *minio.Client
	bucket string
	prefix string
}

func newS3Dl(ctx context.Context, source *IndexSource, shard Shard) (*s3Dl, error) {
	if source.Endpoint == "" {
		return nil, fmt.Errorf("s3 source requires endpoint, e.g. \"s3.amazonaws.com\" or \"play.min.io:9000\"")
	}
	if source.AccessKey == "" || source.SecretKey == "" {
		return nil, fmt.Errorf("s3 source requires access_key and secret_key")
	}

	bucket, prefix, err := parseS3Location(source.Location)
	if err != nil {
		return nil, err
	}

	client, err := minio.New(source.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(source.AccessKey, source.SecretKey, ""),
		Secure: source.UseSSL,
	})
	if err != nil {
		logger.CtxError(ctx, "failed to create minio client", logger.Err(err))
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &s3Dl{
		ctx:    ctx,
		source: source,
		shard:  shard,
		client: client,
		bucket: bucket,
		prefix: prefix,
	}, nil
}

// parseS3Location extracts bucket and prefix from "s3://bucket/prefix" or "bucket/prefix".
func parseS3Location(location string) (bucket, prefix string, err error) {
	loc := strings.TrimPrefix(location, "s3://")
	parts := strings.SplitN(loc, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", "", fmt.Errorf("invalid S3 location: %s", location)
	}
	bucket = parts[0]
	if len(parts) > 1 {
		prefix = parts[1]
	}
	return bucket, prefix, nil
}

// s3Key builds the full S3 object key by joining the prefix with the relative path.
func (d *s3Dl) s3Key(relativePath string) string {
	if d.prefix == "" {
		return relativePath
	}
	return d.prefix + "/" + relativePath
}

func (d *s3Dl) Download() (localPath string, revision int64, err error) {
	log := logger.DefaultLoggerWithCtx(d.ctx).With(
		logger.Interface("source", d.source),
		logger.String("shardKey", d.shard.ShardKey()),
	)

	revision, err = d.readRevision()
	if err != nil {
		log.Error("get revision failed", logger.Err(err))
		return "", 0, config.ErrGetRevisionFailed
	}

	localPath, err = d.downloadShardFile(revision)
	if err != nil {
		log.Error("download shard file failed", logger.Err(err), logger.Int64("revision", revision))
		return "", 0, err
	}

	return localPath, revision, nil
}

func (d *s3Dl) DownloadUpdate(currentRevision int64) (localPath string, revision int64, err error) {
	log := logger.DefaultLoggerWithCtx(d.ctx).With(
		logger.Interface("source", d.source),
		logger.String("shardKey", d.shard.ShardKey()),
	)

	revision, err = d.readRevision()
	if err != nil {
		log.Error("get revision failed", logger.Err(err))
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

	return d.Download()
}

// readRevision reads the _revision file from S3 and parses it as int64.
func (d *s3Dl) readRevision() (int64, error) {
	key := d.s3Key(revisionFile)
	obj, err := d.client.GetObject(d.ctx, d.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return 0, fmt.Errorf("get S3 object %s/%s: %w", d.bucket, key, err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return 0, fmt.Errorf("read S3 object body: %w", err)
	}

	rev, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse revision %q: %w", string(data), err)
	}

	return int64(rev), nil
}

// downloadShardFile downloads the shard index file from S3 to a local temp file
// and returns the local path.
func (d *s3Dl) downloadShardFile(revision int64) (string, error) {
	key := d.s3Key(d.source.GetShardRelPath(revision, d.shard))

	obj, err := d.client.GetObject(d.ctx, d.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("get S3 object %s/%s: %w", d.bucket, key, err)
	}
	defer obj.Close()

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("vectory-s3-%s-%d-*", d.shard.IndexName, revision))
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer tmpFile.Close()

	if _, err = io.Copy(tmpFile, obj); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("write S3 object to temp file: %w", err)
	}

	return tmpFile.Name(), nil
}
