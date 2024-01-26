package json

import (
	"bytes"
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

	serFunc func(event *core.Event) ([]byte, error)

	delim byte
	start []byte
	end   []byte
}

func (s *Json) Init() error {
	switch s.Mode {
	case "jsonl":
		s.delim = '\n'
		s.start = []byte{}
		s.end = []byte{}
	case "array":
		s.delim = ','
		s.start = []byte{'['}
		s.end = []byte{']'}
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

func (s *Json) Self() any {
	return s
}

func (s *Json) Close() error {
	return nil
}

func (s *Json) Serialize(events ...*core.Event) ([]byte, error) {
	now := time.Now()
	buf := bytes.NewBuffer(make([]byte, 0, 4096))
	var result []byte
	buf.Write(s.start)

	lastIter := len(events) - 1
	for i, e := range events {
		rawData, err := s.serFunc(e)
		if err != nil {
			s.Observe(metrics.EventFailed, time.Since(now))
			s.Log.Error("serialization failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.AddTag("::json_serialization_failed")
			if s.OmitFailed {
				now = time.Now()
				continue
			}
			return nil, err
		}
		buf.Write(rawData)
		buf.WriteByte(s.delim)

		if i == lastIter && buf.Len() > 0 {
			buf.Truncate(buf.Len() - 1) // trim last delimeter
			buf.Write(s.end)
			result = make([]byte, buf.Len())
			copy(result, buf.Bytes())
		}

		s.Observe(metrics.EventAccepted, time.Since(now))
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
