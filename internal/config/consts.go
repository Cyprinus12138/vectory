package config

const (
	ConfKeyServiceMeta = "service"
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
