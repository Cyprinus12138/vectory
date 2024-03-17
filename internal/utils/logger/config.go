package logger

import (
	"errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

const DefaultLogLevel = zapcore.InfoLevel

type Config struct {
	Level           string            `json:"level" yaml:"level"`
	DefaultPath     string            `json:"default_path" yaml:"default_path"`
	LevelPath       map[string]string `json:"level_path" yaml:"level_path"`
	LogKafkaEnabled map[string]bool   `json:"log_kafka_enabled" yaml:"log_kafka_enabled"` // TODO
}

func enableDFuncBuilder(lv zapcore.Level) func(lvl zapcore.Level) bool {
	return func(lvl zapcore.Level) bool {
		return lvl == lv
	}
}

func init() {
	logger, _ = zap.NewDevelopment(zap.AddCallerSkip(1))
}

func Init(cfg *Config) (err error) {
	if cfg == nil {
		return errors.New("logger config empty")
	}
	zapCores := make([]zapcore.Core, 0, 0)
	logLevel := DefaultLogLevel
	_ = logLevel.UnmarshalText([]byte(cfg.Level))

	defaultOut, err := os.OpenFile(cfg.DefaultPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}

	for i := logLevel; i < zapcore.InvalidLevel; i++ {
		var out = defaultOut
		if path, ok := cfg.LevelPath[i.String()]; ok {
			out, err = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
			if err != nil {
				out = defaultOut
			}
		}
		zapCores = append(zapCores, zapcore.NewCore(
			zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
			zapcore.Lock(out),
			zap.LevelEnablerFunc(enableDFuncBuilder(i)),
		))
	}

	core := zapcore.NewTee(zapCores...)

	logger = zap.New(core, zap.AddCallerSkip(1))
	defer logger.Sync()
	logger.Info("logger init successfully", Interface("config", cfg))
	return nil
}
