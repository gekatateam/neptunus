package through

import (
	"log/slog"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Through struct {
	alias string
	pipe  string
	in    <-chan *core.Event
	out   chan<- *core.Event
	log   *slog.Logger
}

func (p *Through) Init(_ map[string]any, alias, pipeline string, log *slog.Logger) error {
	p.alias = alias
	p.pipe = pipeline
	p.log = log

	return nil
}

func (p *Through) SetChannels(
	in <-chan *core.Event,
	out chan<- *core.Event,
) {
	p.in = in
	p.out = out
}

func (p *Through) Close() error {
	return nil
}

func (p *Through) Run() {
	for e := range p.in {
		now := time.Now()
		p.out <- e
		metrics.ObserveProcessorSummary("through", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("through", func() core.Processor {
		return &Through{}
	})
}
