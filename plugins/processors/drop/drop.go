package drop

import (
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/elog"
)

type Drop struct {
	*core.BaseProcessor `mapstructure:"-"`
}

func (p *Drop) Init() error {
	return nil
}

func (p *Drop) Close() error {
	return nil
}

func (p *Drop) Run() {
	var now time.Time
	for e := range p.In {
		now = time.Now()
		p.Drop <- e
		p.Log.Debug("event dropped",
			elog.EventGroup(e),
		)
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("drop", func() core.Processor {
		return &Drop{}
	})
}
