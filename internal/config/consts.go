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
	ErrWrongDimension        = errors.New("wrong input dimension")
	ErrEmptyInput            = errors.New("empty input vector")
	ErrNilIndex              = errors.New("index is not loaded")
	ErrInvalidIndexType      = errors.New("invalid index type")
	ErrInvalidScheduleType   = errors.New("invalid schedule type")
	ErrAlreadyReloading      = errors.New("index is reloading")
	ErrIndexRevisionUpToDate = errors.New("index is up-to-date")
)
