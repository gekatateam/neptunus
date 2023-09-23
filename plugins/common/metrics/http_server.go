package metrics

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpServerSummary         *prometheus.SummaryVec
	httpServerSummaryRegister = &sync.Once{}
)

func init() {
	httpServerSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "plugin_http_server_requests_seconds",
			Help:       "Incoming http requests stats.",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001, 1.0: 0},
		},
		[]string{"pipeline", "plugin_name", "address", "uri", "method", "status"},
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

type httpMiddleware struct {
	pipeline   string
	pluginName string
	address    string
}

func NewHttpMiddleware(pipeline string, pluginName string, address string) *httpMiddleware {
	httpServerSummaryRegister.Do(func() { prometheus.MustRegister(httpServerSummary) })

	return &httpMiddleware{
		pipeline:   pipeline,
		pluginName: pluginName,
		address:    address,
	}
}

func (m *httpMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		begin := time.Now()

		s := &statusRecorder{
			ResponseWriter: w,
			Status:         http.StatusOK,
		}

		next.ServeHTTP(s, r)

		httpServerSummary.WithLabelValues(
			m.pipeline, m.pluginName, m.address, r.URL.Path, r.Method, strconv.Itoa(s.Status),
		).Observe(float64(time.Since(begin)) / float64(time.Second))
	})
}
