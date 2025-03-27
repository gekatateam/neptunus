package metrics

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpServerMetricsRegister = &sync.Once{}
	httpServerRequestsSummary *prometheus.SummaryVec
)

func init() {
	httpServerRequestsSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "plugin_http_server_requests_seconds",
			Help:       "Incoming http requests stats.",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"pipeline", "plugin_name", "uri", "method", "status"},
	)
}

type statusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

func HttpServerMiddleware(pipeline string, pluginName string, pathsConfigured bool, next http.Handler) http.Handler {
	httpServerMetricsRegister.Do(func() {
		prometheus.MustRegister(httpServerRequestsSummary)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		begin := time.Now()

		s := &statusRecorder{
			ResponseWriter: w,
			Status:         http.StatusOK,
		}

		next.ServeHTTP(s, r)

		var path string
		if pathsConfigured {
			path = r.Pattern
		} else {
			path = r.URL.Path
		}

		httpServerRequestsSummary.WithLabelValues(
			pipeline, pluginName, path, r.Method, strconv.Itoa(s.Status),
		).Observe(floatSeconds(begin))
	})
}
