package mixer

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/elog"
)

var outs = &sync.Map{}

type outputChans struct {
	mu *sync.RWMutex
	c  *atomic.Int32
	ch []chan<- *core.Event
}

type Mixer struct {
	*core.BaseProcessor `mapstructure:"-"`
	id                  uint64
	line                int
	in                  <-chan *core.Event
}

func (p *Mixer) Init() error {
	return nil
}

func (p *Mixer) SetId(id uint64) {
	p.id = id
}

func (p *Mixer) SetLine(line int) {
	p.line = line
}

func (p *Mixer) Close() error {
	curr, ok := outs.Load(p.id)
	if !ok {
		return nil
	}

	if curr.(outputChans).c.Add(-1) == 0 {
		outs.Delete(p.id)
	}
	return nil
}

func (p *Mixer) SetChannels(in <-chan *core.Event, out chan<- *core.Event, _ chan<- *core.Event) {
	p.in = in
	curr, ok := outs.Load(p.id)
	if !ok {
		stored := outputChans{
			mu: &sync.RWMutex{},
			c:  &atomic.Int32{},
			ch: []chan<- *core.Event{out},
		}
		stored.c.Add(1)
		outs.Store(p.id, stored)
		return
	}

	stored := curr.(outputChans)
	stored.mu.Lock()
	defer stored.mu.Unlock()

	stored.c.Add(1)
	stored.ch = append(stored.ch, out)

	outs.Store(p.id, stored)
}

func (p *Mixer) Run() {
	for e := range p.in {
		now := time.Now()

		p.Log.Debug(fmt.Sprintf("event consumed from chan %v", p.line),
			elog.EventGroup(e),
		)

		curr, _ := outs.Load(p.id)
		stored := curr.(outputChans)
		stored.mu.RLock()

		cases := make([]reflect.SelectCase, 0, len(stored.ch))
		for _, out := range stored.ch {
			cases = append(cases, reflect.SelectCase{
				Dir:  reflect.SelectSend,
				Chan: reflect.ValueOf(out),
				Send: reflect.ValueOf(e),
			})
		}

		stored.mu.RUnlock()
		chosen, _, _ := reflect.Select(cases)
		p.Log.Debug(fmt.Sprintf("event sent to chan %v", chosen),
			elog.EventGroup(e),
		)

		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}
