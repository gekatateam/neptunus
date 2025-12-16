package fanout

import (
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
)

type FanOut struct {
	*core.BaseCore
	in   <-chan *core.Event
	outs []chan<- *core.Event
}

func New(c *core.BaseCore) *FanOut {
	return &FanOut{
		BaseCore: c,
	}
}

func (c *FanOut) SetChannels(in <-chan *core.Event, outs []chan<- *core.Event) {
	c.in = in
	c.outs = outs
}

func (c *FanOut) Run() {
	var now time.Time
	for e := range c.in {
		now = time.Now()
		for i, out := range c.outs {
			if i == len(c.outs)-1 { // send origin event to last consumer
				out <- e
			} else {
				cloned := e.Clone()
				cloned.UUID = e.UUID // keep origin uuid for easier tracing
				out <- cloned
			}
		}
		c.Observe(metrics.EventAccepted, time.Since(now))
	}
}
