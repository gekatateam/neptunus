package metrics

import (
	"fmt"
	"sync"

	"github.com/gekatateam/neptunus/metrics"
)

var (
	pluginCacheSize = func(d cacheDescriptor) string {
		return fmt.Sprintf("plugin_cache_size{pipeline=%q,plugin_name=%q,items=%q}", d.pipeline, d.pluginName, d.cacheItems)
	}
)

type sizer interface {
	Size() int
}

var (
	cacheMetricsRegister  = &sync.Once{}
	cacheMetricsCollector = &cacheCollector{
		stats: make(map[cacheDescriptor]sizer),
		mu:    &sync.Mutex{},
	}
)

type cacheDescriptor struct {
	pipeline   string
	pluginName string
	cacheItems string
}

type cacheCollector struct {
	stats map[cacheDescriptor]sizer
	mu    *sync.Mutex
}

func (c *cacheCollector) append(d cacheDescriptor, sc sizer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stats[d] = sc
}

func (c *cacheCollector) delete(d cacheDescriptor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.stats, d)
	metrics.PluginsSet.UnregisterMetric(pluginCacheSize(d))
}

func (c *cacheCollector) Collect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for d, sc := range c.stats {
		metrics.PluginsSet.GetOrCreateGauge(pluginCacheSize(d), nil).Set(float64(sc.Size()))
	}
}

func RegisterCache(pipeline, pluginName, items string, sc sizer) {
	cacheMetricsRegister.Do(func() {
		metrics.GlobalCollectorsRunner.Append(cacheMetricsCollector)
	})

	cacheMetricsCollector.append(cacheDescriptor{
		pipeline:   pipeline,
		pluginName: pluginName,
		cacheItems: items,
	}, sc)
}

func UnregisterCache(pipeline, pluginName, items string) {
	cacheMetricsCollector.delete(cacheDescriptor{
		pipeline:   pipeline,
		pluginName: pluginName,
		cacheItems: items,
	})
}
