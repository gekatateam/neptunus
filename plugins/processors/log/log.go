package log

import (
	"encoding/json"
	"fmt"

	"github.com/mitchellh/mapstructure"

	"github.com/gekatateam/pipeline/core"
	"github.com/gekatateam/pipeline/logger"
	"github.com/gekatateam/pipeline/plugins"
)

type Log struct {
	Level string `mapstructure:"level"`
	log   logger.Logger
}

func New(config map[string]any, log logger.Logger) (core.Processor, error) {
	l := Log{
		log: log,
	}
	return &l, mapstructure.Decode(config, &l)
}

func (p *Log) Init() error {
	switch p.Level {
	case "trace", "debug", "info", "warn":
		return nil
	default:
		return fmt.Errorf("forbidden logging level :%v; expected one of: trace, debug, info, warn", p.Level)
	}
}

func (p *Log) Process(e ...*core.Event) []*core.Event {
	for _, v := range e {
		event, err := json.Marshal(v)
		if err != nil {
			p.log.Errorf("json marshal failed: %v", err.Error())
		}

		switch p.Level {
		case "trace":
			p.log.Trace(string(event))
		case "debug":
			p.log.Debug(string(event))
		case "info":
			p.log.Info(string(event))
		case "warn":
			p.log.Warn(string(event))
		}
	}
	return e
}

func (p *Log) Close() error {
	return nil
}

func init() {
	plugins.AddProcessor("log", New)
}
