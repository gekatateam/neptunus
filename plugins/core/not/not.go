package not

import (
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
)

type Not struct {
	*core.BaseFilter `mapstructure:"-"`
	core.Filter
}

func (n *Not) SetChannels(in <-chan *core.Event, rejected chan<- *core.Event, accepted chan<- *core.Event) {
	observeFunc := n.BaseFilter.Obs
	n.BaseFilter.Obs = func(plugin, name, pipeline string, status metrics.EventStatus, t time.Duration) {
		switch status {
		case metrics.EventAccepted:
			observeFunc(plugin, name, pipeline, metrics.EventRejected, t)
		case metrics.EventRejected:
			observeFunc(plugin, name, pipeline, metrics.EventAccepted, t)
		default:
			observeFunc(plugin, name, pipeline, status, t)
		}
	}

	n.Filter.SetChannels(in, accepted, rejected) // reverse the channels, so that accepted events are sent to rejected and vice versa
}
