package json

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/goccy/go-json"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Json struct {
	*core.BaseSerializer `mapstructure:"-"`
	DataOnly             bool   `mapstructure:"data_only"`
	OmitFailed           bool   `mapstructure:"omit_failed"`
	Mode                 string `mapstructure:"mode"`

	serFunc func(events ...*core.Event) ([]byte, error)

	delim string
	start string
	end   string
}

func (s *Json) Init() error {
	switch s.Mode {
	case "jsonl":
		s.delim = "\n"
	case "array":
		s.delim = ","
		s.start = "["
		s.end = "]"
	default:
		return fmt.Errorf("forbidden mode: %v, expected one of: jsonl, array", s.Mode)
	}

	if s.OmitFailed {
		s.serFunc = s.serializeTryAll
	} else {
		s.serFunc = s.serializeFailFast
	}

	return nil
}

func (s *Json) Close() error {
	return nil
}

func (s *Json) Serialize(events ...*core.Event) ([]byte, error) {
	return s.serFunc(events...)
}

func (s *Json) serializeTryAll(events ...*core.Event) ([]byte, error) {
	now := time.Now()
	buf := bytes.NewBuffer(make([]byte, 0, 4096))
	buf.WriteString(s.start)

	for _, e := range events {
		rawData, err := s.eventOrData(e)
		if err != nil {
			s.Log.Error("serialization failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.AddTag("::json_serialization_failed")
			s.Observe(metrics.EventFailed, time.Since(now))
			now = time.Now()
			continue
		}
		buf.Write(rawData)
		buf.WriteString(s.delim)

		s.Observe(metrics.EventAccepted, time.Since(now))
		now = time.Now()
	}

	if buf.Len() > len(s.start) { // at least one event serialized successfully
		buf.Truncate(buf.Len() - 1) // trim last delimeter
		buf.WriteString(s.end)
		return buf.Bytes(), nil
	}

	return nil, errors.New("all events serialization failed")
}

func (s *Json) serializeFailFast(events ...*core.Event) ([]byte, error) {
	now := time.Now()
	buf := bytes.NewBuffer(make([]byte, 0, 4096))
	buf.WriteString(s.start)

	var sErr error
	last := len(events) - 1
	for i, e := range events {
		if sErr != nil {
			s.Log.Error("one of previous serialization failed",
				"error", sErr,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.AddTag("::json_serialization_failed")
			s.Observe(metrics.EventFailed, time.Since(now))
			now = time.Now()
			continue
		}

		rawData, err := s.eventOrData(e)
		if err != nil {
			s.Log.Error("serialization failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.AddTag("::json_serialization_failed")
			s.Observe(metrics.EventFailed, time.Since(now))
			now = time.Now()
			sErr = err
			continue
		}
		buf.Write(rawData)
		buf.WriteString(s.delim)

		if i == last && buf.Len() > 0 {
			buf.Truncate(buf.Len() - 1) // trim last delimeter
			buf.WriteString(s.end)
		}

		s.Observe(metrics.EventAccepted, time.Since(now))
		now = time.Now()
	}

	if sErr != nil {
		return nil, sErr
	}
	return buf.Bytes(), nil
}

func (s *Json) eventOrData(event *core.Event) ([]byte, error) {
	if s.DataOnly {
		return json.MarshalNoEscape(event.Data)
	}
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
