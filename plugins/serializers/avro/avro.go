package avro

import (
	"errors"
	"time"

	"github.com/iskorotkov/avro/v2"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/elog"
)

type Avro struct {
	*core.BaseSerializer `mapstructure:"-"`
	Schema               string `mapstructure:"schema"`

	schema avro.Schema
}

func (s *Avro) Init() error {
	schema, err := avro.Parse(s.Schema)
	if err != nil {
		return err
	}
	s.schema = schema

	return nil
}

func (s *Avro) Close() error {
	return nil
}

func (s *Avro) Serialize(events ...*core.Event) ([]byte, error) {
	now := time.Now()
	err := errors.New("avro serializer accepts exactly one event per call")

	if len(events) == 0 {
		return nil, nil
	}

	if len(events) != 1 {
		for _, e := range events {
			s.Log.Error("event serialization failed",
				"error", err,
				elog.EventGroup(e),
			)
			s.Observe(metrics.EventFailed, time.Since(now))
			now = time.Now()
		}
		return nil, err
	}

	result, err := avro.Marshal(s.schema, events[0].Data)
	if err != nil {
		s.Log.Error("event serialization failed",
			"error", err,
			elog.EventGroup(events[0]),
		)
		s.Observe(metrics.EventFailed, time.Since(now))
	} else {
		s.Observe(metrics.EventAccepted, time.Since(now))
	}

	return result, err
}

func init() {
	plugins.AddSerializer("avro", func() core.Serializer {
		return &Avro{}
	})
}
