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

func Int64(key string, val int64) zap.Field {
	return zap.Int64(key, val)
}

// Add necessary field definition below.
