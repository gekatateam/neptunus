package log

import (
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

	logFunc func(args ...interface{})

	in  <-chan *core.Event
	log logger.Logger
	ser core.Serializer
}

func (o *Log) Init(config map[string]any, alias, pipeline string, log logger.Logger) error {
	if err := mapstructure.Decode(config, o); err != nil {
		return err
	}

	o.alias = alias
	o.pipe = pipeline
	o.log = log

	switch o.Level {
	case "trace":
		o.logFunc = o.log.Trace
	case "debug":
		o.logFunc = o.log.Debug
	case "info":
		o.logFunc = o.log.Info
	case "warn":
		o.logFunc = o.log.Warn
	default:
		return fmt.Errorf("forbidden logging level: %v; expected one of: trace, debug, info, warn", o.Level)
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
			o.log.Errorf("serialization failed: %v", err.Error())
			metrics.ObserveOutputSummary("log", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
			continue
		}

		o.logFunc(string(event))
		metrics.ObserveOutputSummary("log", o.alias, o.pipe, metrics.EventAccepted, time.Since(now))
	}
}

func (o *Log) Close() error {
	return o.ser.Close()
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
