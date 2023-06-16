package log

import (
	"fmt"
	"time"

	"github.com/goccy/go-json"

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
	out chan<- *core.Event
	log logger.Logger
}

func New(config map[string]any, alias, pipeline string, log logger.Logger) (core.Processor, error) {
	l := &Log{
		Level: "info",

		log:   log,
		alias: alias,
		pipe:  pipeline,
	}
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
		now := time.Now()
		event, err := json.Marshal(e)
		if err != nil {
			p.log.Errorf("json marshal failed: %v", err.Error())
			e.StackError(fmt.Errorf("log processor: json marshal failed: %v", err.Error()))
			metrics.ObserveProcessorSummary("log", p.alias, p.pipe, metrics.EventFailed, time.Since(now))
			continue
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

		p.out <- e
		metrics.ObserveProcessorSummary("log", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
	}
}

func (p *Log) Close() error {
	return nil
}

func (p *Log) Alias() string {
	return p.alias
}

func init() {
	plugins.AddProcessor("log", New)
}
