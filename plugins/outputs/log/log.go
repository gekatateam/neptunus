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
	*core.BaseOutput `mapstructure:"-"`
	Level            string `mapstructure:"level"`

	logFunc func(msg string, args ...any)

	ser core.Serializer
}

func (o *Log) Init() error {
	switch o.Level {
	case "debug":
		o.logFunc = o.Log.Debug
	case "info":
		o.logFunc = o.Log.Info
	case "warn":
		o.logFunc = o.Log.Warn
	default:
		return fmt.Errorf("forbidden logging level: %v; expected one of: debug, info, warn", o.Level)
	}

	return nil
}

func (o *Log) SetSerializer(s core.Serializer) {
	o.ser = s
}

func (o *Log) Run() {
	for e := range o.In {
		now := time.Now()
		event, err := o.ser.Serialize(e)
		if err != nil {
			o.Log.Error("serialization failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			o.Done <- e
			o.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		o.logFunc(string(event),
			slog.Group("event",
				"id", e.Id,
				"key", e.RoutingKey,
			))
		o.Done <- e
		o.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func (o *Log) Close() error {
	return nil
}

func init() {
	plugins.AddOutput("log", func() core.Output {
		return &Log{
			Level: "info",
		}
	})
}
