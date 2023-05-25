package broadcast

import (
	"time"

	"github.com/gekatateam/pipeline/core"
	"github.com/gekatateam/pipeline/metrics"
)

type Broadcast struct {
	alias string
	pipe  string
	in    <-chan *core.Event
	outs  []chan<- *core.Event
}

func New(alias, pipeline string) *Broadcast {
	return &Broadcast{
		alias: alias,
		pipe:  pipeline,
	}
}

func (c *Broadcast) Init(in <-chan *core.Event, outs []chan<- *core.Event) {
	c.in = in
	c.outs = outs
}

func (c *Broadcast) Run() {
	for e := range c.in {
		now := time.Now()
		for i, out := range c.outs {
			if i == len(c.outs)-1 { // send origin event to last consumer
				out <- e
			} else {
				out <- e.Clone()
			}
		}
		metrics.ObserveCoreSummary("broadcast", c.alias, c.pipe, metrics.EventAccepted, time.Since(now))
	}
}
