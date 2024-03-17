package monitor

import (
	"github.com/Cyprinus12138/vectory/pkg"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"sync"
)

var (
	initOnce  sync.Once
	counter   *prometheus.CounterVec
	errorCode *prometheus.CounterVec
	latency   *prometheus.HistogramVec
	// Customized metrics please register here and init at InitMonitor.
)

// InitMonitor initialize the metrics, please remember to init your customized metrics here.
func InitMonitor() {
	initOnce.Do(func() {
		counter = prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: NameSpace,
			Subsystem: pkg.ClusterName,
			Name:      MetricCount,
		}, []string{"scenario", "key"})
		errorCode = prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: NameSpace,
			Subsystem: pkg.ClusterName,
			Name:      MetricErrorCode,
		}, []string{"scenario", "key", "code"})
		latency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: NameSpace,
			Subsystem: pkg.ClusterName,
			Name:      MetricLatency,
			Buckets:   []float64{8, 16, 32, 64, 128, 256, 384, 512, 640, 768, 1024},
		}, []string{"scenario", "key"})

		prometheus.MustRegister(counter)
		prometheus.MustRegister(errorCode)
		prometheus.MustRegister(latency)
	})
}

func Handler() http.Handler {
	mux := http.NewServeMux()
	if _, err := prometheus.DefaultGatherer.Gather(); err != nil {
		panic(err)
	}
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}

func HandleFunc() gin.HandlerFunc {
	if _, err := prometheus.DefaultGatherer.Gather(); err != nil {
		panic(err)
	}
	return func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	}
}
