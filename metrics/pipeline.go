package metrics

import "github.com/prometheus/client_golang/prometheus"

type pipelineCollectror struct {
	statFunc func() map[string]struct {
		State int
		Lines int
	}
}

func (c *pipelineCollectror) set(f func() map[string]struct {
	State int
	Lines int
}) {
	c.statFunc = f
}

func (c *pipelineCollectror) Describe(ch chan<- *prometheus.Desc) {
	ch <- pipeState
	ch <- pipeLines
}

func (c *pipelineCollectror) Collect(ch chan<- prometheus.Metric) {
	for id, state := range c.statFunc() {
		ch <- prometheus.MustNewConstMetric(
			pipeState,
			prometheus.GaugeValue,
			float64(state.State),
			id,
		)

		ch <- prometheus.MustNewConstMetric(
			pipeLines,
			prometheus.GaugeValue,
			float64(state.Lines),
			id,
		)
	}
}
