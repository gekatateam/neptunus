package protobuf

import (
	"errors"
	"time"

	"github.com/gekatateam/protomap"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/elog"
)

type Protobuf struct {
	*core.BaseSerializer `mapstructure:"-"`
	ProtoFiles           []string `mapstructure:"proto_files"`
	Message              string   `mapstructure:"message"`

	mapper *protomap.Mapper
}

func (s *Protobuf) Init() error {
	if len(s.ProtoFiles) == 0 {
		return errors.New("at least one .proto file required")
	}

	if len(s.Message) == 0 {
		return errors.New("message required")
	}

	mapper, err := protomap.NewMapper(nil, s.ProtoFiles...)
	if err != nil {
		return err
	}

	s.mapper = mapper

	return nil
}

func (s *Protobuf) Close() error {
	return nil
}

func (s *Protobuf) Serialize(events ...*core.Event) ([]byte, error) {
	now := time.Now()

	if len(events) == 0 {
		return nil, errors.New("protobuf serializer accepts exactly one event per call")
	}

	if len(events) != 1 {
		err := errors.New("protobuf serializer accepts only one event per call")
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

	data, ok := events[0].Data.(map[string]any)
	if !ok {
		err := errors.New("event data must be a map")
		s.Log.Error("event serialization failed",
			"error", err,
			elog.EventGroup(events[0]),
		)
		s.Observe(metrics.EventFailed, time.Since(now))
		return nil, err
	}

	return s.mapper.Encode(data, s.Message)
}

func init() {
	plugins.AddSerializer("protobuf", func() core.Serializer {
		return &Protobuf{}
	})
}
