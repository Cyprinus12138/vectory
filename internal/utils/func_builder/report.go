package func_builder

import (
	"context"
	"time"

	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	"github.com/Cyprinus12138/vectory/internal/utils/monitor"
)

func BuildElapsedFunc(monitorName string) func(context.Context, string, bool) func() {
	return func(ctx context.Context, name string, isDebug bool) func() {
		start := time.Now()
		return func() {
			costTime := time.Since(start)
			if isDebug {
				logger.CtxDebug(ctx, "process completed", logger.String("name", name), logger.Duration("duration", costTime))
			} else {
				logger.CtxInfo(ctx, "process completed", logger.String("name", name), logger.Duration("duration", costTime))
			}
			monitor.LatencyObserve(monitorName, name, float64(costTime)/1000)
		}
	}
}

func BuildReportFunc(monitorName string) func(context.Context, string, string, *error) func() {
	return func(ctx context.Context, name string, logNameToAdd string, err *error) func() {
		start := time.Now()
		return func() {
			costTime := time.Since(start)
			logName := name + ":" + logNameToAdd
			if err == nil || *err == nil {
				logger.CtxInfo(ctx, "process completed", logger.String("name", logName), logger.Duration("duration", costTime))
			} else {
				logger.CtxWarn(ctx, "process completed", logger.String("name", logName), logger.Duration("duration", costTime), logger.Err(*err))
				monitor.CountObserveInc(monitorName, name+"Error")
				monitor.CountObserveInc(monitorName, name+":"+(*err).Error())
			}
			monitor.LatencyObserve(monitorName, name, float64(costTime))
			monitor.CountObserveInc(monitorName, name)
		}
	}
}
