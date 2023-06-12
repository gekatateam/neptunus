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

func New(_ map[string]any, alias, pipeline string, log logger.Logger) (core.Parser, error) {
	return &Json{
		alias: alias,
		pipe:  pipeline,
		log:   log,
	}, nil
}

func (p *Json) Alias() string {
	return p.alias
}

func (p *Json) Parse(data []byte, routingKey string) ([]*core.Event, error) {
	now := time.Now()
	events := []*core.Event{}

	if data[0] == '[' { // array provided - [{...},{...},...]
		eventData := []core.Map{}
		if err := json.Unmarshal(data, &eventData); err != nil {
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
		if err := json.Unmarshal(data, &eventData); err != nil {
			metrics.ObserveParserSummary("json", p.alias, p.pipe, metrics.EventFailed, time.Since(now))
			return nil, err
		}
		events = append(events, core.NewEventWithData(routingKey, eventData))
		metrics.ObserveParserSummary("json", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
	}
	return events, nil
}

func init() {
	plugins.AddParser("json", New)
}
