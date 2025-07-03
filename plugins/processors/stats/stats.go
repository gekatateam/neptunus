package stats

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/convert"
)

type Stats struct {
	*core.BaseProcessor `mapstructure:"-"`
	Period              time.Duration       `mapstructure:"period"`
	Mode                string              `mapstructure:"mode"`
	RoutingKey          string              `mapstructure:"routing_key"`
	WithLabels          []string            `mapstructure:"with_labels"`
	WithoutLabels       []string            `mapstructure:"without_labels"`
	Buckets             []float64           `mapstructure:"buckets"`
	DropOrigin          bool                `mapstructure:"drop_origin"`
	Fields              map[string][]string `mapstructure:"fields"`

	id         uint64
	cache      cache
	fields     map[string]metricStats
	buckets    map[float64]float64
	exLabels   map[string]struct{}
	labelsFunc func(e *core.Event) ([]metricLabel, bool)
}

func (p *Stats) Init() error {
	p.fields = make(map[string]metricStats, len(p.Fields))
	p.buckets = make(map[float64]float64, len(p.Buckets)+1)
	p.exLabels = make(map[string]struct{}, len(p.WithoutLabels))
	p.WithLabels = slices.Compact(p.WithLabels)

	for _, v := range p.WithoutLabels {
		p.exLabels[v] = struct{}{}
	}

	if p.Period < time.Second {
		p.Period = time.Second
	}

	p.buckets[math.MaxFloat64] = 0 // +Inf bucket
	for _, v := range p.Buckets {
		p.buckets[v] = 0
	}

	if len(p.WithLabels) > 0 && len(p.WithoutLabels) > 0 {
		return errors.New("with_labels and without_labels set, but only one can be used at the same time")
	}

	p.labelsFunc = p.noLabels

	if len(p.WithLabels) > 0 {
		p.labelsFunc = p.withLabels
	}

	if len(p.WithoutLabels) > 0 {
		p.labelsFunc = p.withoutLabels
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
			if p.DropOrigin {
				p.Drop <- e
			} else {
				p.Out <- e
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

		if m.Stats.Histogram {
			for le, bucket := range m.Value.Buckets {
				e := core.NewEvent(p.RoutingKey)
				e.Timestamp = now

				e.SetLabel("::type", "metric")
				e.SetLabel("::name", m.Descr.Name)
				if le == math.MaxFloat64 {
					e.SetLabel("le", "+Inf")
				} else {
					e.SetLabel("le", strconv.FormatFloat(le, 'f', -1, 64))
				}

				for _, label := range m.Descr.Labels {
					e.SetLabel(label.Key, label.Value)
				}

				e.SetField("stats.bucket", bucket)

				ch <- e
			}
		}
	})
}

func (p *Stats) Observe(e *core.Event) {
	// it is okay if multiple stats will store one label set
	// because there is no race condition
	labels, ok := p.labelsFunc(e)
	if !ok {
		return
	}

	for field, stats := range p.fields {
		f, err := e.GetField(field)
		if err != nil {
			continue // if event has no field, skip it
		}

		fv, err := convert.AnyToFloat(f)
		if err != nil {
			continue // if field is not a number, skip it
		}

		m := &metric{
			Stats: stats,
			Descr: metricDescr{
				Name:   field,
				Labels: labels, // previously saved labels shares here
			},
			Value: metricValue{
				Buckets: maps.Clone(p.buckets),
			},
		}

		p.cache.observe(m, fv)
	}
}

func (p *Stats) noLabels(e *core.Event) ([]metricLabel, bool) {
	return make([]metricLabel, 0), true
}

func (p *Stats) withLabels(e *core.Event) ([]metricLabel, bool) {
	labels := make([]metricLabel, 0, len(p.WithLabels))
	for _, k := range p.WithLabels {
		v, ok := e.GetLabel(k)
		if !ok {
			return nil, false
		}

		labels = append(labels, metricLabel{
			Key:   k,
			Value: v,
		})
	}

	return labels, true
}

func (p *Stats) withoutLabels(e *core.Event) ([]metricLabel, bool) {
	labels := make([]metricLabel, 0, len(e.Labels))
	for k, v := range e.Labels {
		if _, ok := p.exLabels[k]; ok {
			continue
		}

		labels = append(labels, metricLabel{
			Key:   k,
			Value: v,
		})
	}

	slices.SortFunc(labels, compareLabels)
	return labels, true
}

func init() {
	plugins.AddProcessor("stats", func() core.Processor {
		return &Stats{
			Period:     time.Minute,
			RoutingKey: "neptunus.generated.metric",
			Mode:       "shared",
			Buckets:    []float64{0.1, 0.3, 0.5, 0.7, 1.0, 2.0, 5.0, 10.0},
		}
	})
}
