package drop

import (
	"github.com/gekatateam/pipeline/core"
	"github.com/gekatateam/pipeline/logger"
	"github.com/gekatateam/pipeline/metrics"
	"github.com/gekatateam/pipeline/plugins"
)

type Drop struct {
	alias string
	pipe  string
	in    <-chan *core.Event
	log   logger.Logger
}

func New(_ map[string]any, alias, pipeline string, log logger.Logger) (core.Processor, error) {
	return &Drop{
		log:   log, 
		alias: alias,
		pipe:  pipeline,
	}, nil
}

func (p *Drop) Init(
	in <-chan *core.Event,
	_ chan<- *core.Event,
) {
	p.in = in
}

func (p *Drop) Close() error {
	return nil
}

func (p *Drop) Alias() string {
	return p.alias
}

func (p *Drop) Process() {
	for range p.in {
		metrics.ObserveProcessorSummary("drop", p.alias, p.pipe, metrics.EventAccepted, 0)
		continue
	}
}

func init() {
	plugins.AddProcessor("drop", New)
}
