package line

import (
	"fmt"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Rk struct {
	*core.BaseProcessor `mapstructure:"-"`
	Mapping             map[string][]string `mapstructure:"mapping"`
	index               map[string]string
}

func (p *Rk) Init() error {
	p.index = make(map[string]string)
	for newKey, oldKeys := range p.Mapping {
		for _, oldKey := range oldKeys {
			if dup, exists := p.index[oldKey]; exists {
				return fmt.Errorf("duplicate key replacement found for: %v; already indexed: %v, duplicate: %v", oldKey, dup, newKey)
			}

			p.index[oldKey] = newKey
		}
	}

	return nil
}

func (p *Rk) Close() error {
	return nil
}

func (p *Rk) Run() {
	for e := range p.In {
		now := time.Now()

		if newKey, ok := p.index[e.RoutingKey]; ok {
			e.RoutingKey = newKey
		}

		p.Out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("rk", func() core.Processor {
		return &Rk{}
	})
}
