package json

import (
	"fmt"
	"time"

	"github.com/goccy/go-json"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
)

type Json struct {
	alias      string
	pipe       string
	DataOnly   bool   `mapstructure:"data_only"`
	OmitFailed bool   `mapstructure:"omit_failed"`
	Mode       string `mapstructure:"mode"`

	log     logger.Logger
	serFunc func(event *core.Event) ([]byte, error)
	delim   byte
}

func (s *Json) Init(config map[string]any, alias, pipeline string, log logger.Logger) error {
	if err := mapstructure.Decode(config, s); err != nil {
		return err
	}

	s.alias = alias
	s.pipe = pipeline
	s.log = log

	switch s.Mode {
	case "jsonl":
		s.delim = '\n'
	case "array":
		s.delim = ','
	default:
		return fmt.Errorf("forbidden mode: %v, expected one of: jsonl, array", s.Mode)
	}

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

func (s *Json) Serialize(events ...*core.Event) ([]byte, error) {
	now := time.Now()
	var result []byte

	for i, e := range events {
		rawData, err := s.serFunc(e)
		if err != nil {
			metrics.ObserveSerializerSummary("json", s.alias, s.pipe, metrics.EventFailed, time.Since(now))
			s.log.Errorf("serialization failed for event %v: %v", e.Id, err)
			if !s.OmitFailed {
				return nil, err
			}
			now = time.Now()
			continue
		}
		result = append(result, rawData...)
		result = append(result, s.delim)

		if i == len(events)-1 {
			result = result[:len(result)-1] // trim last delimeter
			if s.Mode == "array" {
				result = append([]byte{'['}, result...)
				result = append(result, ']')
			}
		}

		metrics.ObserveSerializerSummary("json", s.alias, s.pipe, metrics.EventAccepted, time.Since(now))
		now = time.Now()
	}

	return result, nil
}

func (s *Json) serializeData(event *core.Event) ([]byte, error) {
	return json.MarshalNoEscape(event.Data)
}

func (s *Json) serializeEvent(event *core.Event) ([]byte, error) {
	return json.MarshalNoEscape(event)
}

func init() {
	plugins.AddSerializer("json", func() core.Serializer {
		return &Json{
			DataOnly:   false,
			Mode:       "jsonl",
			OmitFailed: true,
		}
	})
}
