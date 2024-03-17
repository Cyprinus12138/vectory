package monitor

import (
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
)

func CountObserveInc(scenario, key string) {
	counter.With(prometheus.Labels{"scenario": scenario, "key": key}).Inc()
}

func ErrCodeObserve(scenario, key string, code uint32) {
	errorCode.With(prometheus.Labels{"scenario": scenario, "key": key, "code": strconv.Itoa(int(code))}).Inc()
}

func LatencyObserve(scenario, key string, duration float64) {
	latency.With(prometheus.Labels{"scenario": scenario, "key": key}).Observe(duration)
}
