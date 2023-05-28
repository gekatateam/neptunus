package through

import (
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Through struct {
	alias string
	pipe  string
	in    <-chan *core.Event
	out   chan<- *core.Event
	log   logger.Logger
}

func New(_ map[string]any, alias, pipeline string, log logger.Logger) (core.Processor, error) {
	return &Through{
		log:   log, 
		alias: alias,
		pipe:  pipeline,
	}, nil
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
		metrics.ObserveProcessorSummary("through", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("through", New)
}
