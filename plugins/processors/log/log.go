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
	out chan<- *core.Event
	log logger.Logger
}

func New(config map[string]any, log logger.Logger) (core.Processor, error) {
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

func (p *Log) Init(
	in <-chan *core.Event,
	out chan<- *core.Event,
) {
	p.in = in
	p.out = out
}

func (p *Log) Process() {
	for e := range p.in {
		event, err := json.Marshal(e)
		if err != nil {
			p.log.Errorf("json marshal failed: %v", err.Error())
			e.StackError(fmt.Errorf("log processor: json marshal failed: %v", err.Error()))
			goto EVENT_LOGGED
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

	EVENT_LOGGED:
		p.out <- e
	}
}

func (p *Log) Close() error {
	return nil
}

func init() {
	plugins.AddProcessor("log", New)
}
