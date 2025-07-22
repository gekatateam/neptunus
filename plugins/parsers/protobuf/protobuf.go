package protobuf

import (
	"errors"
	"time"

	"github.com/gekatateam/protomap"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Protobuf struct {
	*core.BaseParser `mapstructure:"-"`
	ProtoFiles       []string `mapstructure:"proto_files"`
	Message          string   `mapstructure:"message"`

	mapper *protomap.Mapper
}

func (p *Protobuf) Init() error {
	if len(p.ProtoFiles) == 0 {
		return errors.New("at least one .proto file required")
	}

	if len(p.Message) == 0 {
		return errors.New("message required")
	}

	mapper, err := protomap.NewMapper(nil, p.ProtoFiles...)
	if err != nil {
		return err
	}

	p.mapper = mapper

	return nil
}

func (p *Protobuf) Close() error {
	return nil
}

func (p *Protobuf) Parse(data []byte, routingKey string) ([]*core.Event, error) {
	now := time.Now()

	eventData, err := p.mapper.Decode(data, p.Message)
	if err != nil {
		p.Observe(metrics.EventFailed, time.Since(now))
		return nil, err
	}

	event := core.NewEventWithData(routingKey, eventData)
	p.Observe(metrics.EventAccepted, time.Since(now))

	return []*core.Event{event}, nil
}

func init() {
	plugins.AddParser("protobuf", func() core.Parser {
		return &Protobuf{}
	})
}
