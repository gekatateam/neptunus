package mixer

import (
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
	for e := range p.ins[atomic.AddInt32(&p.index, 1)] {
		now := time.Now()
		p.out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))
	}

	if atomic.AddInt32(&p.index, -1) == 0 {
		close(p.out)
		p.Log.Debug("output channel closed")
	}
}
