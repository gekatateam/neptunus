package drop

import (
	"github.com/gekatateam/pipeline/core"
	"github.com/gekatateam/pipeline/logger"
	"github.com/gekatateam/pipeline/plugins"
)

type Drop struct {
	in  <-chan *core.Event
	log logger.Logger
}

func New(_ map[string]any, log logger.Logger) (core.Processor, error) {
	l := Drop{
		log: log,
	}
	return &l, nil
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

func (p *Drop) Process() {
	for range p.in {
		continue
	}
}

func init() {
	plugins.AddProcessor("drop", New)
}
