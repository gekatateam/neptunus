package log

import (
	"github.com/gekatateam/pipeline/core"
	"github.com/gekatateam/pipeline/logger"
	"github.com/gekatateam/pipeline/plugins"
)

type ProcessAll struct {
	in  <-chan *core.Event
	out chan<- *core.Event
	log logger.Logger
}

func New(config map[string]any, log logger.Logger) (core.Processor, error) {
	l := ProcessAll{
		log: log,
	}
	return &l, nil
}

func (p *ProcessAll) Init(
	in <-chan *core.Event,
	out chan<- *core.Event,
) {
	p.in = in
	p.out = out
}

func (p *ProcessAll) Close() error {
	return nil
}

func (p *ProcessAll) Process() {
	for e := range p.in {
		p.out <- e
	}
}

func init() {
	plugins.AddProcessor("processall", New)
}
