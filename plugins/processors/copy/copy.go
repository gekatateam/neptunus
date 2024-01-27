package copy

import (
	"errors"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Copy struct {
	*core.BaseProcessor `mapstructure:"-"`
	RoutingKey          string            `mapstructure:"routing_key"`
	SaveTimestamp       bool              `mapstructure:"save_timestamp"`
	Labels              map[string]string `mapstructure:"labels"`
}

func (p *Copy) Init() error {
	if len(p.RoutingKey) == 0 {
		return errors.New("routing key required")
	}

	return nil
}

func (p *Copy) Close() error {
	return nil
}

func (p *Copy) Run() {
	for e := range p.In {
		now := time.Now()

		copy := e.Clone()
		copy.RoutingKey = p.RoutingKey
		for k, v := range p.Labels {
			copy.SetLabel(k, v)
		}

		if p.SaveTimestamp {
			copy.Timestamp = e.Timestamp
		}

		p.Out <- e
		p.Out <- copy
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("copy", func() core.Processor {
		return &Copy{}
	})
}
