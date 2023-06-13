package json

import (
	"time"

	"github.com/goccy/go-json"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
)

type Json struct {
	alias    string
	pipe     string
	DataOnly bool   `mapstructure:"data_only"`

	log     logger.Logger
	serFunc func(event *core.Event) ([]byte, error)
}

func New(config map[string]any, alias, pipeline string, log logger.Logger) (core.Serializer, error) {
	s := &Json{
		alias:    alias,
		pipe:     pipeline,
		DataOnly: false,
		log:      log,
	}

	if err := mapstructure.Decode(config, s); err != nil {
		return nil, err
	}

	if s.DataOnly {
		s.serFunc = s.serializeData
	} else {
		s.serFunc = s.serializeEvent
	}

	return s, nil
}

func (s *Json) Alias() string {
	return s.alias
}

func (s *Json) Serialize(event *core.Event) ([]byte, error) {
	return s.serFunc(event)
}

func (s *Json) serializeData(event *core.Event) ([]byte, error) {
	now := time.Now()

	rawData, err := json.MarshalNoEscape(event.Data)
	if err != nil {
		metrics.ObserveSerializerSummary("json", s.alias, s.pipe, metrics.EventFailed, time.Since(now))
		return nil, err
	}
	metrics.ObserveSerializerSummary("json", s.alias, s.pipe, metrics.EventAccepted, time.Since(now))

	return rawData, nil
}

func (s *Json) serializeEvent(event *core.Event) ([]byte, error) {
	now := time.Now()

	rawData, err := json.MarshalNoEscape(event)
	if err != nil {
		metrics.ObserveSerializerSummary("json", s.alias, s.pipe, metrics.EventFailed, time.Since(now))
		return nil, err
	}
	metrics.ObserveSerializerSummary("json", s.alias, s.pipe, metrics.EventAccepted, time.Since(now))

	return rawData, nil
}

func init() {
	plugins.AddSerializer("json", New)
}
