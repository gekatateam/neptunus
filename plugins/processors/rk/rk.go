package line

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
)

type Rk struct {
	alias   string
	pipe    string
	Mapping map[string][]string `mapstructure:"mapping"`
	index   map[string]string

	in  <-chan *core.Event
	out chan<- *core.Event
	log *slog.Logger
}

func (p *Rk) Init(config map[string]any, alias, pipeline string, log *slog.Logger) error {
	if err := mapstructure.Decode(config, p); err != nil {
		return err
	}

	p.alias = alias
	p.pipe = pipeline
	p.log = log

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

func (p *Rk) SetChannels(
	in <-chan *core.Event,
	out chan<- *core.Event,
) {
	p.in = in
	p.out = out
}

func (p *Rk) Close() error {
	return nil
}

func (p *Rk) Run() {
	for e := range p.in {
		now := time.Now()

		if newKey, ok := p.index[e.RoutingKey]; ok {
			e.RoutingKey = newKey
		}

		p.out <- e
		metrics.ObserveProcessorSummary("rk", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("rk", func() core.Processor {
		return &Rk{}
	})
}
