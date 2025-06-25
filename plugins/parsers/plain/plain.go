package plain

import (
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Plain struct {
	*core.BaseParser `mapstructure:"-"`
	Field            string `mapstructure:"field"`
	AsString         bool   `mapstructure:"as_string"`
}

func (p *Plain) Init() error {
	return nil
}

func (p *Plain) Close() error {
	return nil
}

func (p *Plain) Parse(data []byte, routingKey string) ([]*core.Event, error) {
	now := time.Now()

	event := core.NewEvent(routingKey)
	if p.AsString {
		event.SetField(p.Field, string(data))
	} else {
		event.SetField(p.Field, data)
	}

	p.Observe(metrics.EventAccepted, time.Since(now))

	return []*core.Event{event}, nil
}

func init() {
	plugins.AddParser("plain", func() core.Parser {
		return &Plain{
			Field:    "event",
			AsString: true,
		}
	})
}
