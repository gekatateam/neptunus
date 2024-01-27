package json

import (
	"time"

	"github.com/goccy/go-json"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Json struct {
	*core.BaseParser `mapstructure:"-"`
}


func (p *Json) Init() error {
	return nil
}

func (p *Json) Close() error {
	return nil
}

func (p *Json) Parse(data []byte, routingKey string) ([]*core.Event, error) {
	now := time.Now()
	events := []*core.Event{}

	if data[0] == '[' { // array provided - [{...},{...},...]
		eventData := []core.Map{}
		if err := json.UnmarshalNoEscape(data, &eventData); err != nil {
			p.Observe(metrics.EventFailed, time.Since(now))
			return nil, err
		}

		for _, e := range eventData {
			events = append(events, core.NewEventWithData(routingKey, e))
			p.Observe(metrics.EventAccepted, time.Since(now))
			now = time.Now()
		}
	} else { // object provided - {...}
		eventData := core.Map{}
		if err := json.UnmarshalNoEscape(data, &eventData); err != nil {
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
		return &Json{}
	})
}
