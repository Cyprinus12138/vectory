package config

import "github.com/Cyprinus12138/vectory/internal/utils/config_manager"

type ClusterMetaConfig struct {
	ClusterName string             `json:"cluster_name" yaml:"cluster_name"`
	GrpcEnabled bool               `json:"grpc_enabled" yaml:"grpc_enabled"`
	Admin       ServiceAdminConfig `json:"admin" yaml:"admin"`
	ClusterMode ClusterSetting     `json:"cluster_mode" yaml:"cluster_mode"`
}

type ClusterSetting struct {
	Enabled     bool     `json:"enabled" yaml:"enabled"`
	GracePeriod int      `json:"grace_period" yaml:"grace_period"`
	Endpoints   []string `json:"endpoints" yaml:"etcd_endpoints"`
	Ttl         int64    `json:"ttl" yaml:"ttl"`
}

type ServiceAdminConfig struct {
	EnableMetrics       bool   `json:"enable_metrics" yaml:"enable_metrics"`
	EnablePprof         bool   `json:"enable_pprof" yaml:"enable_pprof"`
	HealthCheckEndpoint string `json:"health_check_endpoint" yaml:"health_check_endpoint"`
}

func GetClusterMetaConfig() *ClusterMetaConfig {
	v := config_manager.Get(ConfKeyClusterMeta)
	if conf, ok := v.(*ClusterMetaConfig); ok {
		return conf
	}
	return nil
}
