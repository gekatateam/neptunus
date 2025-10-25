package metrics

import (
	"fmt"
)

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

type pipelinesCollector struct {
	statFunc func() []PipelineStats
}

func (pc *pipelinesCollector) Collect() {
	for _, pipe := range pc.statFunc() {
		PipelinesSet.GetOrCreateGauge(
			fmt.Sprintf("pipeline_run{pipeline=%q}", pipe.Pipeline),
			nil,
		).Set(bool2float(pipe.Run))
		PipelinesSet.GetOrCreateGauge(
			fmt.Sprintf("pipeline_state{pipeline=%q}", pipe.Pipeline),
			nil,
		).Set(float64(pipe.State))
		PipelinesSet.GetOrCreateGauge(
			fmt.Sprintf("pipeline_processors_lines{pipeline=%q}", pipe.Pipeline),
			nil,
		).Set(float64(pipe.Lines))

		for _, ch := range pipe.Chans {
			PipelinesSet.GetOrCreateGauge(
				fmt.Sprintf("pipeline_channel_capacity{pipeline=%q,plugin=%q,name=%q,desc=%q}", pipe.Pipeline, ch.Plugin, ch.Name, ch.Descriptor),
				nil,
			).Set(float64(ch.Capacity))
			PipelinesSet.GetOrCreateGauge(
				fmt.Sprintf("pipeline_channel_length{pipeline=%q,plugin=%q,name=%q,desc=%q}", pipe.Pipeline, ch.Plugin, ch.Name, ch.Descriptor),
				nil,
			).Set(float64(ch.Length))
		}
	}
}

func CollectPipes(statFunc func() []PipelineStats) {
	GlobalCollectorsRunner.Append(&pipelinesCollector{statFunc: statFunc})
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
