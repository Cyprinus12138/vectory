package config_manager

import (
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
)

const (
	MetaConfigKey  = "config_manager"
	ConfigRootPath = "config"
)

type configManager struct {
	localViper  *viper.Viper
	remoteViper *viper.Viper
}

type MetaConfig struct {
	EnableRemote bool `json:"enable_remote" yaml:"enable_remote"`
	Remote       []struct {
		Provider string `json:"provider" yaml:"provider"`
		Endpoint string `json:"endpoint" yaml:"endpoint"`
		Path     string `json:"path" yaml:"path"`
	} `json:"remote" yaml:"remote"`
}

var cfg *configManager

func Init(cfgPath string, configSchema []viper.RegisteredConfig) (err error) {
	cfg = &configManager{
		localViper: viper.New(),
	}
	configSchema = append(configSchema, viper.RegisteredConfig{
		Key:      MetaConfigKey,
		Schema:   &MetaConfig{},
		CanBeNil: false,
	})
	cfg.localViper.SetName("local")
	cfg.localViper.SetConfigFile(cfgPath)
	cfg.localViper.Register(configSchema)
	err = cfg.localViper.ReadInConfig(true)
	if err != nil {
		return err
	}
	if meta := getMetaConfig(); meta != nil && meta.EnableRemote {
		cfg.remoteViper = viper.New()
		cfg.remoteViper.SetName("remote")
		cfg.remoteViper.SetConfigType("json")
		cfg.remoteViper.Register(configSchema)
		for _, provider := range meta.Remote {
			err = cfg.remoteViper.AddRemoteProvider(provider.Provider, provider.Endpoint, provider.Path)
			if err != nil {
				return err
			}
		}
		err = cfg.remoteViper.ReadRemoteConfig()
		if err != nil {
			return err
		}
		err = cfg.remoteViper.WatchRemoteConfigOnChannel()
		if err != nil {
			return err
		}
	}
	return nil
}

func Get(key string) interface{} {
	if conf := cfg.localViper.Get(key); conf != nil {
		return conf
	} else if cfg.remoteViper != nil {
		return cfg.remoteViper.Get(key)
	}
	return nil
}

func GetString(key string) string {
	if conf := cfg.localViper.GetString(key); conf != "" {
		return conf
	} else if cfg.remoteViper != nil {
		return cfg.remoteViper.GetString(key)
	}
	return ""
}

func Set(key string, val interface{}) {
	cfg.localViper.Set(key, val)
	return
}

func getMetaConfig() *MetaConfig {
	v := cfg.localViper.Get(MetaConfigKey)
	if conf, ok := v.(*MetaConfig); ok {
		return conf
	}
	return nil
}
