package not

import "github.com/gekatateam/neptunus/core"

type Not struct {
	*core.BaseFilter `mapstructure:"-"`
	core.Filter
}

func (n *Not) SetChannels(in <-chan *core.Event, rejected chan<- *core.Event, accepted chan<- *core.Event) {
	n.Filter.SetChannels(in, accepted, rejected) // reverse the channels, so that accepted events are sent to rejected and vice versa
}
