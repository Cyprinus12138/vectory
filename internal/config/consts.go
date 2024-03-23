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
)

var (
	ErrWrongInputDimension   = errors.New("wrong input dimension")
	ErrEmptyInput            = errors.New("empty input vector")
	ErrNilIndex              = errors.New("index is not loaded")
	ErrInvalidIndexType      = errors.New("invalid index type")
	ErrInvalidScheduleType   = errors.New("invalid schedule type")
	ErrAlreadyReloading      = errors.New("index is reloading")
	ErrIndexRevisionUpToDate = errors.New("index is up-to-date")
	ErrUnknownStatus         = errors.New("unknown node status")
	ErrNodeNotAvailable      = errors.New("node under unavailable status")
	ErrTypeAssertion         = errors.New("type assertion failed")
	ErrIndexNotFound         = errors.New("index not found")
	ErrShardNotFound         = errors.New("shard not found")
	ErrShardFileNotFound     = errors.New("shard file not found")
	ErrGetRevisionFailed     = errors.New("get index revision failed")
	ErrOutputDimension       = errors.New("error output dimension")
)
