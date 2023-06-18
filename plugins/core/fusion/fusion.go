package fusion

import (
	"sync"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
)

type Fusion struct {
	alias string
	pipe  string
	wg    *sync.WaitGroup
	ins   []<-chan *core.Event
	out   chan<- *core.Event
}

func New(alias, pipeline string) *Fusion {
	return &Fusion{
		alias: alias,
		pipe:  pipeline,
		wg:    &sync.WaitGroup{},
	}
}

func (c *Fusion) Prepare(ins []<-chan *core.Event, out chan<- *core.Event) {
	c.ins = ins
	c.out = out
}

func (c *Fusion) Run() {
	for _, inputCh := range c.ins {
		c.wg.Add(1)
		go func(ch <-chan *core.Event) {
			for e := range ch {
				now := time.Now()
				c.out <- e
				metrics.ObserveCoreSummary("fusion", c.alias, c.pipe, metrics.EventAccepted, time.Since(now))
			}
			c.wg.Done()
		}(inputCh)
	}
	c.wg.Wait()
}
