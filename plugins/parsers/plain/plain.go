package plain

import (
	"log/slog"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
)

type Plain struct {
	alias string
	pipe  string
	Field string `mapstructure:"field"`
	log   *slog.Logger
}

func (p *Plain) Init(config map[string]any, alias, pipeline string, log *slog.Logger) error {
	if err := mapstructure.Decode(config, p); err != nil {
		return err
	}

	p.alias = alias
	p.pipe = pipeline
	p.log = log

	return nil
}

func (p *Plain) Alias() string {
	return p.alias
}

func (p *Plain) Close() error {
	return nil
}

func (p *Plain) Parse(data []byte, routingKey string) ([]*core.Event, error) {
	now := time.Now()

	event := core.NewEventWithData(routingKey, core.Map{
		p.Field: string(data),
	})
	metrics.ObserveParserSummary("plain", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))

	return []*core.Event{event}, nil
}

func init() {
	plugins.AddParser("plain", func() core.Parser {
		return &Plain{
			Field: "event",
		}
	})
}
