package json

import (
	"time"

	"github.com/goccy/go-json"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Json struct {
	alias string
	pipe  string
	log   logger.Logger
}

func (p *Json) Init(_ map[string]any, alias, pipeline string, log logger.Logger) error {
	p.alias = alias
	p.pipe = pipeline
	p.log = log

	return nil
}

func (p *Json) Alias() string {
	return p.alias
}

func (p *Json) Parse(data []byte, routingKey string) ([]*core.Event, error) {
	now := time.Now()
	events := []*core.Event{}

	if data[0] == '[' { // array provided - [{...},{...},...]
		eventData := []core.Map{}
		if err := json.UnmarshalNoEscape(data, &eventData); err != nil {
			metrics.ObserveParserSummary("json", p.alias, p.pipe, metrics.EventFailed, time.Since(now))
			return nil, err
		}

		for _, e := range eventData {
			events = append(events, core.NewEventWithData(routingKey, e))
			metrics.ObserveParserSummary("json", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
			now = time.Now()
		}
	} else { // object provided - {...}
		eventData := core.Map{}
		if err := json.UnmarshalNoEscape(data, &eventData); err != nil {
			metrics.ObserveParserSummary("json", p.alias, p.pipe, metrics.EventFailed, time.Since(now))
			return nil, err
		}
		events = append(events, core.NewEventWithData(routingKey, eventData))
		metrics.ObserveParserSummary("json", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
	}
	return events, nil
}

func init() {
	plugins.AddParser("json", func () core.Parser {
		return &Json{}
	})
}
