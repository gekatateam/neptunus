package avro

import (
	"fmt"
	"time"

	"github.com/iskorotkov/avro/v2"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Avro struct {
	*core.BaseParser `mapstructure:"-"`
	Schema           string `mapstructure:"schema"`
	SplitArray       bool   `mapstructure:"split_array"`

	schema  avro.Schema
	topType avro.Type
}

func (p *Avro) Init() error {
	s, err := avro.Parse(p.Schema)
	if err != nil {
		return err
	}
	p.schema = s

	switch s.Type() {
		case avro.Record, avro.Map, avro.Array:
			p.topType = s.Type()
		default:
			return fmt.Errorf("top-level type must be record, map or array, got %v", s.Type())
	}

	return nil
}

func (p *Avro) Close() error {
	return nil
}

func (p *Avro) Parse(data []byte, routingKey string) ([]*core.Event, error) {
	now := time.Now()
	events := []*core.Event{}

	if p.topType == avro.Array {
		body := []any{}
		if err := avro.Unmarshal(p.schema, data, &body); err != nil {
			p.Observe(metrics.EventFailed, time.Since(now))
			return nil, err
		}

		if p.SplitArray {
			for _, e := range body {
				events = append(events, core.NewEventWithData(routingKey, e))
				p.Observe(metrics.EventAccepted, time.Since(now))
				now = time.Now()
			}
		} else {
			events = append(events, core.NewEventWithData(routingKey, body))
			p.Observe(metrics.EventAccepted, time.Since(now))
		}
	} else { // record, map - both here
		body := map[string]any{}
		if err := avro.Unmarshal(p.schema, data, &body); err != nil {
			p.Observe(metrics.EventFailed, time.Since(now))
			return nil, err
		}
		println(body)
		events = append(events, core.NewEventWithData(routingKey, body))
		p.Observe(metrics.EventAccepted, time.Since(now))
	}

	return events, nil
}

func init() {
	plugins.AddParser("avro", func() core.Parser {
		return &Avro{}
	})
}
