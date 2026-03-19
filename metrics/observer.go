package metrics

import (
	"fmt"
	"time"

	"github.com/VictoriaMetrics/metrics"
)

const pluginMetricName = "%v_plugin_processed_events{plugin=%q,name=%q,pipeline=%q,status=%q}"

type PluginDescriptor struct {
	Kind     string
	Plugin   string
	Name     string
	Pipeline string
}

type Observer struct {
	d PluginDescriptor
	m map[EventStatus]*metrics.Summary
}

func NewObserver(desc PluginDescriptor) *Observer {
	return &Observer{
		d: desc,
		m: make(map[EventStatus]*metrics.Summary, 3),
	}
}

func (o *Observer) Observe(status EventStatus, t time.Duration) {
	// long way, first initialization
	// if there is no metric for passed status, we need to create and save it in local map
	// for future updates
	metric, ok := o.m[status]
	if !ok {
		metric = CoreSet.GetOrCreateSummaryExt(
			fmt.Sprintf(pluginMetricName, o.d.Kind, o.d.Plugin, o.d.Name, o.d.Pipeline, status),
			DefaultMetricWindow,
			DefaultSummaryQuantiles,
		)
		o.m[status] = metric
	}

	metric.Update(t.Seconds())
}

// func (o *Observer) Close() error {
// 	for status := range o.m {
// 		CoreSet.UnregisterMetric(fmt.Sprintf(pluginMetricName, o.d.Kind, o.d.Plugin, o.d.Name, o.d.Pipeline, status))
// 	}
// 	return nil
// }
