package metrics

import (
	"fmt"
	"sync"

	"github.com/gekatateam/neptunus/metrics"
)

var (
	pluginStatsMetricsCached = func(d statsDescriptor) string {
		return fmt.Sprintf("plugin_stats_metrics_cached{pipeline=%q,plugin_name=%q}", d.pipeline, d.pluginName)
	}
)

type statsCounter interface {
	StatsCached() int
}

var (
	statsMetricsRegister  = &sync.Once{}
	statsMetricsCollector = &statsCollector{
		stats: make(map[statsDescriptor]statsCounter),
		mu:    &sync.Mutex{},
	}
)

type statsDescriptor struct {
	pipeline   string
	pluginName string
}

type statsCollector struct {
	stats map[statsDescriptor]statsCounter
	mu    *sync.Mutex
}

func (c *statsCollector) append(d statsDescriptor, sc statsCounter) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stats[d] = sc
}

func (c *statsCollector) delete(d statsDescriptor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.stats, d)
	metrics.PluginsSet.UnregisterMetric(pluginStatsMetricsCached(d))
}

func (c *statsCollector) Collect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for d, sc := range c.stats {
		metrics.PluginsSet.GetOrCreateGauge(pluginStatsMetricsCached(d), nil).Set(float64(sc.StatsCached()))
	}
}

func RegisterStatsCache(pipeline, pluginName string, sc statsCounter) {
	statsMetricsRegister.Do(func() {
		metrics.GlobalCollectorsRunner.Append(statsMetricsCollector)
	})

	statsMetricsCollector.append(statsDescriptor{
		pipeline:   pipeline,
		pluginName: pluginName,
	}, sc)
}

func UnregisterStatsCache(pipeline, pluginName string) {
	statsMetricsCollector.delete(statsDescriptor{
		pipeline:   pipeline,
		pluginName: pluginName,
	})
}
