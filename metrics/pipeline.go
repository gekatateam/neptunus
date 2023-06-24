package metrics

import "github.com/prometheus/client_golang/prometheus"

type pipelineCollectror struct {
	statFunc func() map[string]int
}

func (c *pipelineCollectror) set(f func() map[string]int) {
	c.statFunc = f
}

func (c *pipelineCollectror) Describe(ch chan<- *prometheus.Desc) {
	ch <- pipeState
}

func (c *pipelineCollectror) Collect(ch chan<- prometheus.Metric) {
	for id, state := range c.statFunc() {
		ch <- prometheus.MustNewConstMetric(
			pipeState,
			prometheus.GaugeValue,
			float64(state),
			id,
		)
	}
}
