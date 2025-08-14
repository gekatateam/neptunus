package protobuf

import (
	"errors"
	"time"

	"github.com/bufbuild/protocompile"
	"github.com/gekatateam/protomap"
	"github.com/gekatateam/protomap/interceptors"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/elog"
)

type Protobuf struct {
	*core.BaseSerializer `mapstructure:"-"`
	ProtoFiles           []string `mapstructure:"proto_files"`
	ImportPaths          []string `mapstructure:"import_paths"`
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

	compiler := &protocompile.Compiler{
		Resolver: protocompile.CompositeResolver{
			protocompile.WithStandardImports(&protocompile.SourceResolver{}),
			&protocompile.SourceResolver{ImportPaths: s.ImportPaths},
		},
	}

	mapper, err := protomap.NewMapper(compiler, s.ProtoFiles...)
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
	err := errors.New("protobuf serializer accepts exactly one event per call")

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

	result, err := s.mapper.Encode(events[0].Data, s.Message, interceptors.DurationEncoder, interceptors.TimeEncoder)
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
	plugins.AddSerializer("protobuf", func() core.Serializer {
		return &Protobuf{}
	})
}
