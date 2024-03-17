package config

import (
	"github.com/Cyprinus12138/vectory/internal/utils/config_manager"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
)

func GetLoggerConf() *logger.Config {
	v := config_manager.Get(ConfKeyLogger)
	if conf, ok := v.(*logger.Config); ok {
		return conf
	}
	return nil
}
