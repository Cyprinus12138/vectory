package logger

import (
	"go.uber.org/zap"
	"time"
)

func String(key string, val string) zap.Field {
	return zap.String(key, val)
}

func Interface(key string, val interface{}) zap.Field {
	return zap.Any(key, val)
}

func Err(err error) zap.Field {
	return zap.Error(err)
}

func Duration(key string, duration time.Duration) zap.Field {
	return zap.Duration(key, duration)
}

// Add necessary field definition below.
