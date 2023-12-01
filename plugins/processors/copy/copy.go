package copy

import (
	"errors"
	"log/slog"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
)

type Copy struct {
	alias         string
	pipe          string
	RoutingKey    string            `mapstructure:"routing_key"`
	SaveTimestamp bool              `mapstructure:"save_timestamp"`
	Labels        map[string]string `mapstructure:"labels"`

	in  <-chan *core.Event
	out chan<- *core.Event
	log *slog.Logger
}

func (p *Copy) Init(config map[string]any, alias, pipeline string, log *slog.Logger) error {
	if err := mapstructure.Decode(config, p); err != nil {
		return err
	}

	p.alias = alias
	p.pipe = pipeline
	p.log = log

	if len(p.RoutingKey) == 0 {
		return errors.New("routing key required")
	}

	return nil
}

func (p *Copy) Prepare(
	in <-chan *core.Event,
	out chan<- *core.Event,
) {
	p.in = in
	p.out = out
}

func (p *Copy) Close() error {
	return nil
}

func (p *Copy) Alias() string {
	return p.alias
}

func (p *Copy) Run() {
	for e := range p.in {
		now := time.Now()

		copy := e.Clone()
		copy.RoutingKey = p.RoutingKey
		for k, v := range p.Labels {
			copy.AddLabel(k, v)
		}

		if p.SaveTimestamp {
			copy.Timestamp = e.Timestamp
		}

		p.out <- e
		p.out <- copy
		metrics.ObserveProcessorSummary("copy", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("copy", func() core.Processor {
		return &Copy{}
	})
}
