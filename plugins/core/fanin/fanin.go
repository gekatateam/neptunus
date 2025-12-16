package fanin

import (
	"sync"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
)

type FanIn struct {
	*core.BaseCore
	wg  *sync.WaitGroup
	ins []<-chan *core.Event
	out chan<- *core.Event
}

func New(c *core.BaseCore) *FanIn {
	return &FanIn{
		BaseCore: c,
		wg:       &sync.WaitGroup{},
	}
}

func (c *FanIn) SetChannels(ins []<-chan *core.Event, out chan<- *core.Event) {
	c.ins = ins
	c.out = out
}

func (c *FanIn) Run() {
	for _, inputCh := range c.ins {
		c.wg.Add(1)
		go func(ch <-chan *core.Event) {
			var now time.Time
			for e := range ch {
				now = time.Now()
				c.out <- e
				c.Observe(metrics.EventAccepted, time.Since(now))
			}
			c.wg.Done()
		}(inputCh)
	}
	c.wg.Wait()
}
