package plain

import (
	"log/slog"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Plain struct {
	*core.BaseParser `mapstructure:"-"`
	Field            string `mapstructure:"field"`
	log              *slog.Logger
}

func (p *Plain) Init() error {
	return nil
}

func (p *Plain) Close() error {
	return nil
}

func (p *Plain) Parse(data []byte, routingKey string) ([]*core.Event, error) {
	now := time.Now()

	event := core.NewEventWithData(routingKey, map[string]any{
		p.Field: string(data),
	})
	p.Observe(metrics.EventAccepted, time.Since(now))

	return []*core.Event{event}, nil
}

func init() {
	plugins.AddParser("plain", func() core.Parser {
		return &Plain{
			Field: "event",
		}
	})
}
