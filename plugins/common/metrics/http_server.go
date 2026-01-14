package metrics

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gekatateam/neptunus/metrics"
)

type statusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

func HttpServerMiddleware(pipeline string, pluginName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		begin := time.Now()

		s := &statusRecorder{
			ResponseWriter: w,
			Status:         http.StatusOK,
		}

		next.ServeHTTP(s, r)

		path := r.URL.Path
		if len(r.Pattern) > 0 {
			path = r.Pattern
		}

		metrics.PluginsSet.GetOrCreateSummaryExt(
			fmt.Sprintf("plugin_http_server_requests_seconds{pipeline=%q,plugin_name=%q,uri=%q,method=%q,status=%q}", pipeline, pluginName, path, r.Method, strconv.Itoa(s.Status)),
			metrics.DefaultMetricWindow,
			metrics.DefaultSummaryQuantiles,
		).Update(time.Since(begin).Seconds())
	})
}
