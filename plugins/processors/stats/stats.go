package stats

import (
	"fmt"
	"slices"
	"sync"
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
	Interval   time.Duration       `mapstructure:"interval"`
	RoutingKey string              `mapstructure:"routing_key"`
	Labels     []string            `mapstructure:"labels"`
	DropOrigin bool                `mapstructure:"drop_origin"`
	Fields     map[string][]string `mapstructure:"fields"`

	cache  map[uint64]*metric
	fields map[string]metricStats
	mu     *sync.Mutex

	in  <-chan *core.Event
	out chan<- *core.Event
	log logger.Logger
}

func (p *Stats) Init(config map[string]any, alias, pipeline string, log logger.Logger) error {
	if err := mapstructure.Decode(config, p); err != nil {
		return err
	}

	p.alias = alias
	p.pipe = pipeline
	p.log = log
	p.fields = make(map[string]metricStats)
	p.cache = make(map[uint64]*metric)
	p.mu = &sync.Mutex{}
	p.Labels = slices.Compact(p.Labels)

	for k, v := range p.Fields {
		if len(v) == 0 {
			return fmt.Errorf("field %v has no configured stats", k)
		}

		fields := slices.Compact(v)
		for _, v := range fields {
			if ok := statsMap[v]; !ok {
				return fmt.Errorf("unknown stat for field %v: %v", k, v)
			}
		}

		p.fields[k] = stats(fields)
	}

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
		p.log.Info("flushing goroutine started")
		for {
			select {
			case <-stop:
				ticker.Stop()
				p.Flush()
				p.log.Info("flushing goroutine stopped")
				close(done)
				return
			case <-ticker.C:
				p.Flush()
			}
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

	close(stop)
	<-done
}

func (p *Stats) Flush() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for _, m := range p.cache {
		e := core.NewEvent(p.RoutingKey)
		e.Timestamp = now

		e.AddLabel("::type", "metric")
		e.AddLabel("::name", m.Descr.Name)

		for _, label := range m.Descr.Labels {
			e.AddLabel(label.Key, label.Value)
		}

		if m.Stats.Count {
			e.SetField("stats.count", m.Value.Count)
		}

		if m.Stats.Sum {
			e.SetField("stats.sum", m.Value.Sum)
		}

		if m.Stats.Gauge {
			e.SetField("stats.gauge", m.Value.Gauge)
		}

		if m.Stats.Avg {
			e.SetField("stats.avg", m.Value.Avg)
		}

		if m.Stats.Min {
			e.SetField("stats.min", m.Value.Min)
		}

		if m.Stats.Max {
			e.SetField("stats.max", m.Value.Max)
		}

		p.out <- e
		m.reset()
	}
}

func (p *Stats) Observe(e *core.Event) {
	for field, stats := range p.fields {
		f, err := e.GetField(field)
		if err != nil {
			continue // if event has no field, skip it
		}

		fv, ok := convert(f)
		if !ok {
			continue // if field is not a number, skip it
		}

		labels := []metricLabel{}
		for _, k := range p.Labels {
			v, ok := e.GetLabel(k)
			if !ok {
				return // if event has no label, skip it
			}

			labels = append(labels, metricLabel{
				Key:   k,
				Value: v,
			})
		}

		m := &metric{Descr: metricDescr{
			Name:   field,
			Labels: labels,
		}}

		if metric, ok := p.cache[m.hash()]; ok {
			m = metric
		} else { // hit an uncached netric
			m.Stats = stats
			p.cache[m.hash()] = m
		}

		p.mu.Lock()
		m.observe(fv)
		p.mu.Unlock()
	}
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
		return &Stats{
			Interval:   time.Minute,
			RoutingKey: "neptunus.generated.metric",
		}
	})
}
