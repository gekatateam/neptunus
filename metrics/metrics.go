package metrics

import (
	"time"

	"github.com/VictoriaMetrics/metrics"
)

var (
	// CoreSet is a set with plugins core metrics
	CoreSet = metrics.NewSet()
	// PluginsSet is a set with plugins custom metrics
	PluginsSet = metrics.NewSet()
	// PipelinesSet is a set with pipelines metrics
	PipelinesSet = metrics.NewSet()
)

var (
	DefaultMetricCollectInterval = 15 * time.Second
	DefaultMetricWindow          = time.Minute
	DefaultSummaryQuantiles      = []float64{0.5, 0.9, 0.99, 1.0}
)

func init() {
	metrics.ExposeMetadata(true)
}
