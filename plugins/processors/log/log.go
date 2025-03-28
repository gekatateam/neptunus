package log

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Log struct {
	*core.BaseProcessor `mapstructure:"-"`
	Level               string `mapstructure:"level"`

	logFunc func(msg string, args ...any)

	ser core.Serializer
}

func (p *Log) Init() error {
	switch p.Level {
	case "debug":
		p.logFunc = p.Log.Debug
	case "info":
		p.logFunc = p.Log.Info
	case "warn":
		p.logFunc = p.Log.Warn
	default:
		return fmt.Errorf("forbidden logging level: %v; expected one of: debug, info, warn", p.Level)
	}

	return nil
}

func (p *Log) SetSerializer(s core.Serializer) {
	p.ser = s
}

func (p *Log) Run() {
	for e := range p.In {
		now := time.Now()
		event, err := p.ser.Serialize(e)
		if err != nil {
			p.Log.Error("event serialization failed",
				"error", err.Error(),
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.StackError(fmt.Errorf("log processor: event serialization failed: %v", err.Error()))
			p.Out <- e
			p.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		p.logFunc(string(event),
			slog.Group("event",
				"id", e.Id,
				"key", e.RoutingKey,
			),
		)
		p.Out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func (p *Log) Close() error {
	return p.ser.Close()
}

func init() {
	plugins.AddProcessor("log", func() core.Processor {
		return &Log{
			Level: "info",
		}
	})
}
