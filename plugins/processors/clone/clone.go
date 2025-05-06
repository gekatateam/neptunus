package clone

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
	Count               int               `mapstructure:"count"`
	Labels              map[string]string `mapstructure:"labels"`
}

func (p *Clone) Init() error {
	if p.Count <= 0 {
		p.Count = 1
	}
	return nil
}

func (p *Clone) Close() error {
	return nil
}

func (p *Clone) Run() {
	for e := range p.In {
		for range p.Count {
			now := time.Now()

			cloned := e.Clone()
			if len(p.RoutingKey) > 0 {
				cloned.RoutingKey = p.RoutingKey
			}

			for k, v := range p.Labels {
				cloned.SetLabel(k, v)
			}

			if p.SaveTimestamp {
				cloned.Timestamp = e.Timestamp
			}

			p.Out <- e
			p.Out <- cloned
			p.Observe(metrics.EventAccepted, time.Since(now))
		}
	}
}

func init() {
	plugins.AddProcessor("clone", func() core.Processor {
		return &Clone{
			Count: 1,
		}
	})
}
