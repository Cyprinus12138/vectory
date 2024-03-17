package logger

import (
	"context"
	"go.uber.org/zap"
)

var logger *zap.Logger

func DefaultLogger() *zap.Logger {
	return logger
}

func DefaultLoggerWithCtx(ctx context.Context) *zap.Logger {
	if reqId, ok := ctx.Value("ReqId").(string); ok {
		return logger.With(String("ReqId", reqId))
	}
	return logger
}

func Debug(msg string, fields ...zap.Field) {
	DefaultLogger().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	DefaultLogger().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	DefaultLogger().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	DefaultLogger().Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	DefaultLogger().Fatal(msg, fields...)
}

func CtxDebug(ctx context.Context, msg string, fields ...zap.Field) {
	DefaultLoggerWithCtx(ctx).Debug(msg, fields...)
}

func CtxInfo(ctx context.Context, msg string, fields ...zap.Field) {
	DefaultLoggerWithCtx(ctx).Info(msg, fields...)
}

func CtxWarn(ctx context.Context, msg string, fields ...zap.Field) {
	DefaultLoggerWithCtx(ctx).Warn(msg, fields...)
}

func CtxError(ctx context.Context, msg string, fields ...zap.Field) {
	DefaultLoggerWithCtx(ctx).Error(msg, fields...)
}

func CtxFatal(ctx context.Context, msg string, fields ...zap.Field) {
	DefaultLoggerWithCtx(ctx).Fatal(msg, fields...)
}
