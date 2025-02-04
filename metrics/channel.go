package metrics

import "github.com/prometheus/client_golang/prometheus"

type ChanDesc string

const (
	ChanDescIn   ChanDesc = "in"   // input channels for any plugins
	ChanDescDrop ChanDesc = "drop" // channels that drops events, e.g. dropped by processors, input/output filters
	ChanDescDone ChanDesc = "done" // outputs channels for done events
)

type ChanStats struct {
	Capacity   int
	Length     int
	Plugin     string
	Name       string
	Pipeline   string
	Descriptor string
}

type chanCollector struct {
	statFunc []func() ChanStats
}

func (c *chanCollector) append(f func() ChanStats) {
	c.statFunc = append(c.statFunc, f)
}

func (c *chanCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- chanCapacity
	ch <- chanLength
}

func (c *chanCollector) Collect(ch chan<- prometheus.Metric) {
	for _, f := range c.statFunc {
		state := f()
		ch <- prometheus.MustNewConstMetric(
			chanCapacity,
			prometheus.GaugeValue,
			float64(state.Capacity),
			state.Plugin,
			state.Name,
			state.Pipeline,
			state.Descriptor,
		)
		ch <- prometheus.MustNewConstMetric(
			chanLength,
			prometheus.GaugeValue,
			float64(state.Length),
			state.Plugin,
			state.Name,
			state.Pipeline,
			state.Descriptor,
		)
	}
}
