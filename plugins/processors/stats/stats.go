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
	var statsMap = map[string]bool{
		"count": true,
		"sum":   true,
		"gauge": true,
		"avg":   true,
		"min":   true,
		"max":   true,
	}

	p.alias = alias
	p.pipe = pipeline
	p.log = log

	if err := mapstructure.Decode(config, p); err != nil {
		return err
	}

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
	<-done
}

func (p *Stats) Flush() {}

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
		m.observe(fv)
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

func stats(stats []string) metricStats {
	s := metricStats{}
	for _, v := range stats {
		switch v {
		case "sum":
			s.Sum = true
		case "count":
			s.Count = true
		case "gauge":
			s.Gauge = true
		case "avg":
			s.Avg = true
		case "min":
			s.Min = true
		case "max":
			s.Max = true
		}
	}
	return s
}

func init() {
	plugins.AddProcessor("stats", func() core.Processor {
		return &Stats{}
	})
}
