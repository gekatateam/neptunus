package through

import (
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/elog"
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
	for e := range p.In {
		now := time.Now()

		if p.Sleep > 0 {
			p.Log.Warn("sleeping",
				elog.EventGroup(e),
			)
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
