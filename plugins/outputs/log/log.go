package log

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
)

type Log struct {
	alias string
	pipe  string
	Level string `mapstructure:"level"`

	logFunc func(msg string, args ...any)

	in  <-chan *core.Event
	log *slog.Logger
	ser core.Serializer
}

func (o *Log) Init(config map[string]any, alias, pipeline string, log *slog.Logger) error {
	if err := mapstructure.Decode(config, o); err != nil {
		return err
	}

	o.alias = alias
	o.pipe = pipeline
	o.log = log

	switch o.Level {
	case "debug":
		o.logFunc = o.log.Debug
	case "info":
		o.logFunc = o.log.Info
	case "warn":
		o.logFunc = o.log.Warn
	default:
		return fmt.Errorf("forbidden logging level: %v; expected one of: debug, info, warn", o.Level)
	}

	return nil
}

func (o *Log) Prepare(in <-chan *core.Event) {
	o.in = in
}

func (o *Log) SetSerializer(s core.Serializer) {
	o.ser = s
}

func (o *Log) Run() {
	for e := range o.in {
		now := time.Now()
		event, err := o.ser.Serialize(e)
		if err != nil {
			o.log.Error("serialization failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			metrics.ObserveOutputSummary("log", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
			continue
		}

		o.logFunc(string(event))
		metrics.ObserveOutputSummary("log", o.alias, o.pipe, metrics.EventAccepted, time.Since(now))
	}
}

func (o *Log) Close() error {
	return nil
}

func (o *Log) Alias() string {
	return o.alias
}

func init() {
	plugins.AddOutput("log", func() core.Output {
		return &Log{
			Level: "info",
		}
	})
}
