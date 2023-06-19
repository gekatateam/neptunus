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
	DataOnly bool `mapstructure:"data_only"`

	log     logger.Logger
	serFunc func(event *core.Event) ([]byte, error)
}

func (s *Json) Init(config map[string]any, alias, pipeline string, log logger.Logger) error {
	if err := mapstructure.Decode(config, s); err != nil {
		return err
	}

	s.alias = alias
	s.pipe = pipeline
	s.log = log

	if s.DataOnly {
		s.serFunc = s.serializeData
	} else {
		s.serFunc = s.serializeEvent
	}

	return nil
}

func (s *Json) Alias() string {
	return s.alias
}

func (s *Json) Close() error {
	return nil
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
	plugins.AddSerializer("json", func() core.Serializer {
		return &Json{
			DataOnly: false,
		}
	})
}
