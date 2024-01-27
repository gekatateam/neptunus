package through

import (
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Through struct {
	*core.BaseProcessor `mapstructure:"-"`
}

func (p *Through) Init() error {
	return nil
}

func (p *Through) Close() error {
	return nil
}

func (p *Through) Run() {
	for e := range p.In {
		now := time.Now()
		p.Out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("through", func() core.Processor {
		return &Through{}
	})
}
