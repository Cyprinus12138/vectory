package main

import (
	"context"
	"github.com/Cyprinus12138/vectory/internal/app"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/utils/config_manager"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	"github.com/Cyprinus12138/vectory/internal/utils/monitor"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"syscall"
)

var server app.App

func init() {
	err := config_manager.Init("./etc/service.yml", []viper.RegisteredConfig{
		{
			Key:      config.ConfKeyServiceMeta,
			CanBeNil: false,
			Schema:   &config.ServiceMetaConfig{},
		},
		{
			Key:      config.ConfKeyLogger,
			CanBeNil: false,
			Schema:   &logger.Config{},
		},
		{
			Key:      config.ConfKeyEnv,
			CanBeNil: true,
		},
		{
			Key:      config_manager.MetaConfigKey,
			Schema:   &config_manager.MetaConfig{},
			CanBeNil: false,
		},
	})
	if err != nil {
		panic(errors.WithMessage(err, "config init failed"))
	}

	err = logger.Init(config.GetLoggerConf())
	if err != nil {
		panic(errors.WithMessage(err, "logger init failed"))
	}

	err = server.Setup(context.Background())
	if err != nil {
		panic(errors.WithMessage(err, "server setup failed"))
	}

	monitor.InitMonitor()
}

func main() {
	go exitWhenNotified()
	err := server.Start()
	if err != nil {
		panic(err)
	}
}
func exitWhenNotified() {
	terminateSignals := make(chan os.Signal, 1)
	signal.Notify(terminateSignals, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM) //NOTE:: syscall.SIGKILL we cannot catch kill -9 as its force kill signal.
	select {
	case s := <-terminateSignals:
		logger.Info("signal received, graceful stopping", logger.String("signal", s.String()))
		server.Stop(s)
		break //break is not necessary to add here as if server is closed our main function will end.
	}
}
