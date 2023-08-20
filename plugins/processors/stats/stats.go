package stats

import (
	"slices"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
)

type Stats struct {
	alias      string
	pipe       string
	Interval   time.Duration `mapstructure:"interval"`
	RoutingKey string        `mapstructure:"routing_key"`
	Labels     []string      `mapstructure:"labels"`
	DropOrigin bool          `mapstructure:"drop_origin"`

	Count []string `mapstructure:"count"`
	Sum   []string `mapstructure:"sum"`
	Gauge []string `mapstructure:"gauge"`
	Avg   []string `mapstructure:"avg"`
	Min   []string `mapstructure:"min"`
	Max   []string `mapstructure:"max"`

	cache map[uint64]*metric

	in  <-chan *core.Event
	out chan<- *core.Event
	log logger.Logger
}

func (p *Stats) Init(config map[string]any, alias, pipeline string, log logger.Logger) error {
	p.alias = alias
	p.pipe = pipeline
	p.log = log

	if err := mapstructure.Decode(config, p); err != nil {
		return err
	}

	p.Labels = slices.Compact(p.Labels)
	p.Count = slices.Compact(p.Count)
	p.Sum = slices.Compact(p.Sum)
	p.Gauge = slices.Compact(p.Gauge)
	p.Avg = slices.Compact(p.Avg)
	p.Min = slices.Compact(p.Min)
	p.Max = slices.Compact(p.Max)

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
	stop, done := make(chan struct{}), make(chan struct{})
	ticker := time.NewTicker(p.Interval)
	go func() {
		select {
		case <-stop:
			p.Flush()
			close(done)
			return
		case <-ticker.C:
			p.Flush()
		}
	}()

	for e := range p.in {
		now := time.Now()
		p.Observe(e)
		if !p.DropOrigin {
			p.out <- e
		}
		metrics.ObserveProcessorSummary("stats", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
	}

	ticker.Stop()
	close(stop)
	<- done
}

func (p *Stats) Flush() {}

func (p *Stats) Observe(e *core.Event) {
	
}

func convert(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32: 
		return float64(v), true
	case int: 
		return float64(v), true
	case int8: 
		return float64(v), true
	case int16: 
		return float64(v), true
	case int32: 
		return float64(v), true
	case int64: 
		return float64(v), true
	case uint: 
		return float64(v), true
	case uint8: 
		return float64(v), true
	case uint16: 
		return float64(v), true
	case uint32: 
		return float64(v), true
	case uint64: 
		return float64(v), true
	default:
		return 0, false
	}
}

func init() {
	plugins.AddProcessor("stats", func() core.Processor {
		return &Stats{}
	})
}
