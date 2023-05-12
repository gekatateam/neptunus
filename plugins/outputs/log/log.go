package log

import (
	"encoding/json"
	"fmt"

	"github.com/gekatateam/pipeline/core"
	"github.com/gekatateam/pipeline/logger"
	"github.com/gekatateam/pipeline/pkg/mapstructure"
	"github.com/gekatateam/pipeline/plugins"
)

type Log struct {
	Level string `mapstructure:"level"`

	in  <-chan *core.Event
	log logger.Logger
}

func New(config map[string]any, log logger.Logger) (core.Output, error) {
	l := &Log{log: log}
	if err := mapstructure.Decode(config, l); err != nil {
		return nil, err
	}

	switch l.Level {
	case "trace", "debug", "info", "warn":
	default:
		return nil, fmt.Errorf("forbidden logging level: %v; expected one of: trace, debug, info, warn", l.Level)
	}

	return l, nil
}

func (o *Log) Init(in <-chan *core.Event) {
	o.in = in
}

func (o *Log) Listen() {
	for e := range o.in {
		event, err := json.Marshal(e)
		if err != nil {
			o.log.Errorf("json marshal failed: %v", err.Error())
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
	}
}

func (o *Log) Close() error {
	return nil
}

func init() {
	plugins.AddOutput("log", New)
}
