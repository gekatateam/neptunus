package mixer

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
)

type Mixer struct {
	*core.BaseProcessor `mapstructure:"-"`

	ins []<-chan *core.Event
	out chan *core.Event

	index int32
}

func (p *Mixer) Init() error {
	p.index = -1
	return nil
}

func (p *Mixer) Close() error {
	return nil
}

func (p *Mixer) OutChan() chan *core.Event {
	return p.out
}

func (p *Mixer) PutChannels(in <-chan *core.Event, out chan *core.Event) {
	if p.out == nil {
		p.out = out
	}

	p.ins = append(p.ins, in)
}

func (p *Mixer) Run() {
	i := atomic.AddInt32(&p.index, 1)
	for e := range p.ins[i] {
		now := time.Now()
		p.Log.Debug(fmt.Sprintf("event from chan %v consumed", i))
		p.out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))
	}

	if atomic.AddInt32(&p.index, -1) == 0 {
		close(p.out)
		p.Log.Debug("output channel closed")
	}
}
