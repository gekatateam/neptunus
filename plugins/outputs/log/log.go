package log

import (
	"errors"
	"fmt"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
)

type Log struct {
	alias string
	pipe  string
	Level string `mapstructure:"level"`

	in  <-chan *core.Event
	log logger.Logger
	ser core.Serializer
}

func (o *Log) Init(config map[string]any, alias, pipeline string, log logger.Logger) error {
	if err := mapstructure.Decode(config, o); err != nil {
		return err
	}

	if o.ser == nil {
		return errors.New("log output requires serializer plugin")
	}

	switch o.Level {
	case "trace", "debug", "info", "warn":
	default:
		return fmt.Errorf("forbidden logging level: %v; expected one of: trace, debug, info, warn", o.Level)
	}

	o.alias = alias
	o.pipe = pipeline
	o.log = log

	return nil
}

func (o *Log) Prepare(in <-chan *core.Event) {
	o.in = in
}

func (o *Log) SetSerializer(s core.Serializer) {
	o.ser = s
}

func (o *Log) Listen() {
	for e := range o.in {
		now := time.Now()
		event, err := o.ser.Serialize(e)
		if err != nil {
			o.log.Errorf("serialization failed: %v", err.Error())
			metrics.ObserveOutputSummary("log", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
			continue
		}

		switch o.Level {
		case "trace":
			o.log.Trace(string(event))
		case "debug":
			o.log.Debug(string(event))
		case "info":
			o.log.Info(string(event))
		case "warn":
			o.log.Warn(string(event))
		}
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
