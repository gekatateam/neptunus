package metrics

import "github.com/prometheus/client_golang/prometheus"

type ChanDesc string

const (
	ChanIn   ChanDesc = "in"   // input channels for any plugins
	ChanDrop ChanDesc = "drop" // channels that drops events, e.g. dropped by processors, input/output filters
	ChanDone ChanDesc = "done" // outputs channels for done events
)

type ChanStatsFunc func() ChanStats

type ChanStats struct {
	Capacity   int
	Length     int
	Plugin     string
	Name       string
	Descriptor ChanDesc
}

type PipelineStats struct {
	Pipeline string
	Run      bool
	State    int
	Lines    int
	Chans    []ChanStats
}

func CollectPipes(statFunc func() []PipelineStats) {
	pipes.set(statFunc)
}

type pipelineCollectror struct {
	statFunc func() []PipelineStats
}

func (c *pipelineCollectror) set(f func() []PipelineStats) {
	c.statFunc = f
}

func (c *pipelineCollectror) Describe(ch chan<- *prometheus.Desc) {
	ch <- pipeState
	ch <- pipeLines
	ch <- chanCapacity
	ch <- chanLength
}

func (c *pipelineCollectror) Collect(ch chan<- prometheus.Metric) {
	for _, pipe := range c.statFunc() {
		ch <- prometheus.MustNewConstMetric(
			pipeState,
			prometheus.GaugeValue,
			float64(pipe.State),
			pipe.Pipeline,
		)

		ch <- prometheus.MustNewConstMetric(
			pipeRun,
			prometheus.GaugeValue,
			bool2float(pipe.Run),
			pipe.Pipeline,
		)

		ch <- prometheus.MustNewConstMetric(
			pipeLines,
			prometheus.GaugeValue,
			float64(pipe.Lines),
			pipe.Pipeline,
		)

		for _, channel := range pipe.Chans {
			ch <- prometheus.MustNewConstMetric(
				chanCapacity,
				prometheus.GaugeValue,
				float64(channel.Capacity),
				channel.Plugin,
				channel.Name,
				pipe.Pipeline,
				string(channel.Descriptor),
			)
			ch <- prometheus.MustNewConstMetric(
				chanLength,
				prometheus.GaugeValue,
				float64(channel.Length),
				channel.Plugin,
				channel.Name,
				pipe.Pipeline,
				string(channel.Descriptor),
			)
		}
	}
}

// https://github.com/golang/go/issues/6011
func bool2float(cond bool) (x float64) {
	if cond {
		x = 1
	} else {
		x = 0
	}
	return
}
