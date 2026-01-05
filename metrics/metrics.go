package metrics

import (
	"context"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/gekatateam/neptunus/logger"
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
	GlobalCollectorsRunner       = &collectorsRunner{}
	DefaultMetricCollectInterval = 15 * time.Second
	DefaultMetricWindow          = time.Minute
	DefaultSummaryQuantiles      = []float64{0.5, 0.9, 0.99, 1.0}
)

type Collector interface {
	Collect()
}

type collectorsRunner struct {
	collectors []Collector
}

func (cr *collectorsRunner) Append(c Collector) {
	cr.collectors = append(cr.collectors, c)
}

func (cr *collectorsRunner) Run(ctx context.Context, interval time.Duration) {
	metrics.ExposeMetadata(true)
	ticker := time.NewTicker(interval)

	logger.Default.Info("metric collectors runner started")
	defer ticker.Stop()
	defer logger.Default.Info("metric collectors runner exited")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logger.Default.Debug("metric collectors runner - collection cycle started")
			for _, c := range cr.collectors {
				c.Collect()
			}
			logger.Default.Debug("metric collectors runner - collection cycle done")
		}
	}
}
