package through

import (
	"time"

	"github.com/gekatateam/pipeline/core"
	"github.com/gekatateam/pipeline/logger"
	"github.com/gekatateam/pipeline/metrics"
	"github.com/gekatateam/pipeline/plugins"
)

type Through struct {
	alias string
	in    <-chan *core.Event
	out   chan<- *core.Event
	log   logger.Logger
}

func New(_ map[string]any, alias string, log logger.Logger) (core.Processor, error) {
	return &Through{log: log, alias: alias}, nil
}

func (p *Through) Init(
	in <-chan *core.Event,
	out chan<- *core.Event,
) {
	p.in = in
	p.out = out
}

func (p *Through) Close() error {
	return nil
}

func (p *Through) Alias() string {
	return p.alias
}

func (p *Through) Process() {
	for e := range p.in {
		now := time.Now()
		p.out <- e
		metrics.ObserveProcessorSummary("through", p.alias, metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("through", New)
}
