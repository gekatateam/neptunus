package json

import (
	"time"

	"github.com/goccy/go-json"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Json struct{
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

func (p *Json) Parse(data []byte) (*core.Event, error) {
	now := time.Now()
	event := core.NewEvent("")
	
	if err := json.Unmarshal(data, event); err != nil {
		metrics.ObserveParserSummary("json", p.alias, p.pipe, metrics.EventFailed, time.Since(now))
		return nil, err
	}

	metrics.ObserveParserSummary("json", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
	return event, nil
}

func init() {
	plugins.AddParser("json", New)
}
