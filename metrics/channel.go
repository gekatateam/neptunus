package metrics

import "github.com/prometheus/client_golang/prometheus"

type ChanStats struct {
	Capacity int
	Length   int
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
		)
		ch <- prometheus.MustNewConstMetric(
			chanLength,
			prometheus.GaugeValue,
			float64(state.Length),
		)
	}
}
