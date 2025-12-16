package through

import (
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Through struct {
	*core.BaseProcessor `mapstructure:"-"`
	Sleep               time.Duration `mapstructure:"sleep"`
}

func (p *Through) Init() error {
	return nil
}

func (p *Through) Close() error {
	return nil
}

func (p *Through) Run() {
	var now time.Time
	for e := range p.In {
		now = time.Now()

		if p.Sleep > 0 {
			time.Sleep(p.Sleep)
		}

		p.Out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("through", func() core.Processor {
		return &Through{}
	})
}
