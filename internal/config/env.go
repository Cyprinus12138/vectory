package config

import (
	"fmt"
	"github.com/Cyprinus12138/vectory/internal/utils/config_manager"
	"os"
	"strconv"
)

const (
	FmtEnv = "env.%s"

	EnvNamePort    = "PORT"
	EnvNameRpcPort = "RPC_PORT"
	EnvPodIp       = "POD_IP"
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
