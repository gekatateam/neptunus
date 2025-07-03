package mixer

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
)

var _ core.Mixer = (*Mixer)(nil)

type Mixer struct {
	*core.BaseProcessor `mapstructure:"-"`

	ins []<-chan *core.Event
	out chan *core.Event

	ch    int32
	index int
}

func (p *Mixer) Init() error {
	p.ch = -1
	p.index = -1
	return nil
}

func (p *Mixer) Close() error {
	return nil
}

func (p *Mixer) IncrIndex() int {
	p.index++
	return p.index
}

func (p *Mixer) OutChan() chan *core.Event {
	return p.out
}

func (p *Mixer) AppendChannels(in <-chan *core.Event, out chan *core.Event) {
	if p.out == nil {
		p.out = out
	}

	p.ins = append(p.ins, in)
}

func (p *Mixer) Run() {
	i := atomic.AddInt32(&p.ch, 1)
	for e := range p.ins[i] {
		now := time.Now()
		p.Log.Debug(fmt.Sprintf("event from chan %v consumed", i))
		p.out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))
	}

	if atomic.AddInt32(&p.ch, -1) == -1 {
		close(p.out)
		p.Log.Debug("output channel closed")
	}
}
