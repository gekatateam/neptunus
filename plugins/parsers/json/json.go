package json

import (
	"fmt"
	"time"

	std "encoding/json"

	goccy "github.com/goccy/go-json"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Json struct {
	*core.BaseParser `mapstructure:"-"`
	Unmarshaler      string `mapstructure:"unmarshaler"`
	SplitArray       bool   `mapstructure:"split_array"`

	unmarshal func([]byte, any) error
}

func (p *Json) Init() error {
	switch p.Unmarshaler {
	case "standard":
		p.unmarshal = std.Unmarshal
	case "goccy":
		p.unmarshal = goccy.Unmarshal
	default:
		return fmt.Errorf("unknown unmarshaler: %v", p.Unmarshaler)
	}

	return nil
}

func (p *Json) Close() error {
	return nil
}

func (p *Json) Parse(data []byte, routingKey string) ([]*core.Event, error) {
	now := time.Now()
	events := []*core.Event{}

	if len(data) == 0 {
		e := core.NewEvent(routingKey)
		p.Observe(metrics.EventAccepted, time.Since(now))
		return []*core.Event{e}, nil
	}

	if data[0] == '[' { // array provided - [{...},{...},...]
		eventData := []any{}
		if err := p.unmarshal(data, &eventData); err != nil {
			p.Observe(metrics.EventFailed, time.Since(now))
			return nil, err
		}

		if p.SplitArray {
			for _, e := range eventData {
				events = append(events, core.NewEventWithData(routingKey, e))
				p.Observe(metrics.EventAccepted, time.Since(now))
				now = time.Now()
			}
		} else {
			events = append(events, core.NewEventWithData(routingKey, eventData))
			p.Observe(metrics.EventAccepted, time.Since(now))
		}
	} else { // object provided - {...}
		eventData := map[string]any{}
		if err := p.unmarshal(data, &eventData); err != nil {
			p.Observe(metrics.EventFailed, time.Since(now))
			return nil, err
		}
		events = append(events, core.NewEventWithData(routingKey, eventData))
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
	return events, nil
}

func init() {
	plugins.AddParser("json", func() core.Parser {
		return &Json{
			Unmarshaler: "standard",
			SplitArray:  true,
		}
	})
}
