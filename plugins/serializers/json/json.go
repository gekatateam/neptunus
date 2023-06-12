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
	Mode     string `mapstructure:"mode"`
	DataOnly bool   `mapstructure:"data_only"`

	log logger.Logger
}

func New(config map[string]any, alias, pipeline string, log logger.Logger) (core.Serializer, error) {
	s := &Json{
		alias:    alias,
		pipe:     pipeline,
		Mode:     "jsonl",
		DataOnly: false,
		log:      log,
	}

	if err := mapstructure.Decode(config, s); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Json) Alias() string {
	return s.alias
}

func (s *Json) Serialize(event *core.Event) ([]byte, error) {
	now := time.Now()
	rawData := []byte{}
	var err error

	if s.DataOnly {
		rawData, err = json.MarshalNoEscape(event.Data)
		if err != nil {
			metrics.ObserveSerializerSummary("json", s.alias, s.pipe, metrics.EventFailed, time.Since(now))
			return nil, err
		}
	} else {
		rawData, err = json.MarshalNoEscape(event)
		if err != nil {
			metrics.ObserveSerializerSummary("json", s.alias, s.pipe, metrics.EventFailed, time.Since(now))
			return nil, err
		}
	}
	metrics.ObserveSerializerSummary("json", s.alias, s.pipe, metrics.EventAccepted, time.Since(now))

	return rawData, nil
}

func init() {
	plugins.AddSerializer("json", New)
}
