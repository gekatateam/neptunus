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
	out chan<- *core.Event
	log logger.Logger
	ser core.Serializer
}

func (p *Log) Init(config map[string]any, alias, pipeline string, log logger.Logger) error {
	if err := mapstructure.Decode(config, p); err != nil {
		return err
	}

	if p.ser == nil {
		return errors.New("log processor requires serializer plugin")
	}

	switch p.Level {
	case "trace", "debug", "info", "warn":
	default:
		return fmt.Errorf("forbidden logging level: %v; expected one of: trace, debug, info, warn", p.Level)
	}

	p.alias = alias
	p.pipe = pipeline
	p.log = log

	return nil
}

func (p *Log) Prepare(
	in <-chan *core.Event,
	out chan<- *core.Event,
) {
	p.in = in
	p.out = out
}

func (p *Log) SetSerializer (s core.Serializer) {
	p.ser = s
}

func (p *Log) Process() {
	for e := range p.in {
		now := time.Now()
		event, err := p.ser.Serialize(e)
		if err != nil {
			p.log.Errorf("event serialization failed: %v", err.Error())
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
	plugins.AddProcessor("log", func () core.Processor {
		return &Log{
			Level: "info",
		}
	})
}
