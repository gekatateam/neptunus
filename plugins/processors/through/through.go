package through

import (
	"github.com/gekatateam/pipeline/core"
	"github.com/gekatateam/pipeline/logger"
	"github.com/gekatateam/pipeline/plugins"
)

type Through struct {
	in  <-chan *core.Event
	out chan<- *core.Event
	log logger.Logger
}

func New(_ map[string]any, log logger.Logger) (core.Processor, error) {
	return &Through{log: log}, nil
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

func (p *Through) Process() {
	for e := range p.in {
		p.out <- e
	}
}

func init() {
	plugins.AddProcessor("through", New)
}
