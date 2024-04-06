package copy

import (
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Clone struct {
	*core.BaseProcessor `mapstructure:"-"`
	RoutingKey          string            `mapstructure:"routing_key"`
	SaveTimestamp       bool              `mapstructure:"save_timestamp"`
	Labels              map[string]string `mapstructure:"labels"`
}

func (p *Clone) Init() error {
	return nil
}

func (p *Clone) Close() error {
	return nil
}

func (p *Clone) Run() {
	for e := range p.In {
		now := time.Now()

		copy := e.Clone()
		if len(p.RoutingKey) > 0 {
			copy.RoutingKey = p.RoutingKey
		}

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
	plugins.AddProcessor("clone", func() core.Processor {
		return &Clone{}
	})
}
