package config

import "github.com/pkg/errors"

const (
	ConfKeyClusterMeta = "cluster"
	ConfKeyEnv         = "env"
	ConfKeyLogger      = "logger"
)

const (
	FmtAddr = "%s:%d"
)

const (
	NetworkTCP = "tcp"

	BroadcastAddr = "0.0.0.0"

	FmtEtcdSvcPath         = "svc/%s"
	FmtEtcdSvcRegisterPath = "svc/%s/%s"
	FmtEtcdSvcResolveFmt   = "etcd:///svc/%s"
	FmtUrlForResolver      = "%s://%s"
)

var (
	ErrWrongInputDimension    = errors.New("wrong input dimension")
	ErrEmptyInput             = errors.New("empty input vector")
	ErrNilIndex               = errors.New("index is not loaded")
	ErrInvalidIndexType       = errors.New("invalid index type")
	ErrInvalidScheduleType    = errors.New("invalid schedule type")
	ErrAlreadyReloading       = errors.New("index is reloading")
	ErrIndexRevisionUpToDate  = errors.New("index is up-to-date")
	ErrUnknownStatus          = errors.New("unknown node status")
	ErrNodeNotAvailable       = errors.New("node under unavailable status")
	ErrClusterNotInitialised  = errors.New("cluster not initialised hash ring is nil")
	ErrTypeAssertion          = errors.New("type assertion failed")
	ErrIndexNotFound          = errors.New("index not found")
	ErrShardNotFound          = errors.New("shard not found")
	ErrShardFileNotFound      = errors.New("shard file not found")
	ErrReloadConfNotFound     = errors.New("reload config is nil")
	ErrGetRevisionFailed      = errors.New("get index revision failed")
	ErrOutputDimension        = errors.New("error output dimension")
	ErrPartlySuccess          = errors.New("load shards partly success")
	ErrNoSuccess              = errors.New("load shards no success")
	ErrGetManifestsFailed     = errors.New("get manifest failed")
	ErrUnmarshal              = errors.New("unmarshal failed")
	ErrManifestValidateFailed = errors.New("invalid index manifest")
	ErrResolveIndexShard      = errors.New("cannot resolve shardKey")
	ErrResolveNoAvailableNode = errors.New("no available node resolved")
	ErrBadRequest             = errors.New("invalid request body")
)
