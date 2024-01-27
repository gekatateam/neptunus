package fusion

import (
	"sync"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
)

type Fusion struct {
	*core.BaseCore
	wg  *sync.WaitGroup
	ins []<-chan *core.Event
	out chan<- *core.Event
}

func New(c *core.BaseCore) *Fusion {
	return &Fusion{
		BaseCore: c,
		wg:       &sync.WaitGroup{},
	}
}

func (c *Fusion) SetChannels(ins []<-chan *core.Event, out chan<- *core.Event) {
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
				c.Observe(metrics.EventAccepted, time.Since(now))
			}
			c.wg.Done()
		}(inputCh)
	}
	c.wg.Wait()
}
