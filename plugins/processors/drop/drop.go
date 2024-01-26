package drop

import (
	"log/slog"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Drop struct {
	*core.BaseProcessor
}

func (p *Drop) Init() error {
	return nil
}

func (p *Drop) Self() any {
	return p
}

func (p *Drop) Close() error {
	return nil
}

func (p *Drop) Run() {
	for e := range p.BaseProcessor.In {
		now := time.Now()
		e.Done()
		p.BaseProcessor.Log.Debug("event dropped",
			slog.Group("event",
				"id", e.Id,
				"key", e.RoutingKey,
			),
		)
		p.BaseProcessor.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("drop", func() core.Processor {
		return &Drop{}
	})
}
