package stats

import (
	"fmt"
	"slices"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Stats struct {
	id                  uint64
	*core.BaseProcessor `mapstructure:"-"`
	Period              time.Duration       `mapstructure:"period"`
	Mode                string              `mapstructure:"mode"`
	RoutingKey          string              `mapstructure:"routing_key"`
	Labels              []string            `mapstructure:"labels"`
	DropOrigin          bool                `mapstructure:"drop_origin"`
	Fields              map[string][]string `mapstructure:"fields"`

	cache  cache
	fields map[string]metricStats
}

func (p *Stats) Init() error {
	p.fields = make(map[string]metricStats)
	p.Labels = slices.Compact(p.Labels)

	if p.Period < time.Second {
		p.Period = time.Second
	}

	switch p.Mode {
	case "individual":
		p.cache = newIndividualCache()
	case "shared":
		p.cache = newSharedCache(p.id)
	default:
		return fmt.Errorf("unknown mode: %v, expected one of: shared, individual", p.Mode)
	}

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

func (p *Stats) Close() error {
	p.cache.clear()
	return nil
}

func (p *Stats) SetId(id uint64) {
	p.id = id
}

func (p *Stats) Run() {
	ticker := time.NewTicker(p.Period)

	for {
		select {
		case <-ticker.C:
			p.Flush()
		case e, ok := <-p.In:
			if !ok {
				ticker.Stop()
				p.Flush()
				return
			}

			now := time.Now()
			p.Observe(e)
			if !p.DropOrigin {
				p.Out <- e
			} else {
				e.Done()
			}
			p.BaseProcessor.Observe(metrics.EventAccepted, time.Since(now))
		}
	}
}

func (p *Stats) Flush() {
	now := time.Now()

	p.cache.flush(p.Out, func(m *metric, ch chan<- *core.Event) {
		e := core.NewEvent(p.RoutingKey)
		e.Timestamp = now

		e.SetLabel("::type", "metric")
		e.SetLabel("::name", m.Descr.Name)

		for _, label := range m.Descr.Labels {
			e.SetLabel(label.Key, label.Value)
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

		ch <- e
	})
}

func (p *Stats) Observe(e *core.Event) {
	// it is okay if multiple stats will store one label set
	// because there is no race condition
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

	for field, stats := range p.fields {
		f, err := e.GetField(field)
		if err != nil {
			continue // if event has no field, skip it
		}

		fv, ok := convert(f)
		if !ok {
			continue // if field is not a number, skip it
		}

		m := &metric{
			Stats: stats,
			Descr: metricDescr{
				Name:   field,
				Labels: labels, // previously saved labels shares here
			},
		}

		p.cache.observe(m, fv)
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
			Period:     time.Minute,
			RoutingKey: "neptunus.generated.metric",
			Mode:       "shared",
		}
	})
}
