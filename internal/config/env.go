package config

import (
	"fmt"
	"github.com/Cyprinus12138/vectory/internal/utils/config_manager"
	"github.com/Cyprinus12138/vectory/pkg"
	"os"
	"strconv"
)

const (
	FmtEnv = "env.%s"

	EnvNamePort     = "PORT"
	EnvNameRpcPort  = "RPC_PORT"
	EnvPodIp        = "POD_IP"
	EnvIndexPath    = "IDX_PATH"
	EnvEtCDRootPath = "ETCD_ROOT"
)

func GetEnv(envName string) string {
	if env := os.Getenv(envName); len(env) > 0 {
		return env
	}
	return config_manager.GetString(fmt.Sprintf(FmtEnv, envName))
}

func GetPort() int {
	if port, err := strconv.Atoi(GetEnv(EnvNamePort)); err == nil {
		return port
	}
	return 0
}

func GetRpcPort() int {
	if port, err := strconv.Atoi(GetEnv(EnvNameRpcPort)); err == nil {
		return port
	}
	return 0
}

func GetPodIp() string {
	if ip := GetEnv(EnvPodIp); ip != "" {
		return ip
	}
	return BroadcastAddr
}

func GetEnvLocalIndexPath() string {
	if path := GetEnv(EnvIndexPath); path != "" {
		return path
	}
	return DefaultIndexPath
}

func GetEnvEtCDRoot() string {
	if root := GetEnv(EnvEtCDRootPath); root != "" {
		return root
	}
	return fmt.Sprintf("/vectory/%s", pkg.ClusterName)
}
