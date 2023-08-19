package stats

import (
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Stats struct {
	alias      string
	pipe       string
	Window     time.Duration `mapstructure:"window"`
	RoutingKey string        `mapstructure:"routing_key"`
	Labels     []string      `mapstructure:"labels"`

	Count []string `mapstructure:"count"`
	Sum   []string `mapstructure:"sum"`
	Gauge []string `mapstructure:"gauge"`
	Avg   []string `mapstructure:"avg"`
	Min   []string `mapstructure:"min"`
	Max   []string `mapstructure:"max"`

	stats configuredStats
	cache map[uint64]metric

	in  <-chan *core.Event
	out chan<- *core.Event
	log logger.Logger
}

type configuredStats struct {
	Count bool
	Sum   bool
	Gauge bool
	Ang   bool
	Min   bool
	Max   bool
}

func (p *Stats) Init(config map[string]any, alias, pipeline string, log logger.Logger) error {
	p.alias = alias
	p.pipe = pipeline
	p.log = log

	return nil
}

func (p *Stats) Prepare(
	in <-chan *core.Event,
	out chan<- *core.Event,
) {
	p.in = in
	p.out = out
}

func (p *Stats) Close() error {
	return nil
}

func (p *Stats) Alias() string {
	return p.alias
}

func (p *Stats) Run() {
	for e := range p.in {
		now := time.Now()
		p.out <- e
		metrics.ObserveProcessorSummary("stats", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("stats", func() core.Processor {
		return &Stats{}
	})
}
