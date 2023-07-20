package starlark

import (
	"errors"
	"fmt"
	"time"

	"go.starlark.net/starlark"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
)

type Starlark struct {
	alias string
	pipe  string
	Code  string `mapstructure:"code"`
	File  string `mapstructure:"file"`

	thread *starlark.Thread
	stFunc starlark.Value

	in  <-chan *core.Event
	out chan<- *core.Event
	log logger.Logger
}

func (p *Starlark) Init(config map[string]any, alias, pipeline string, log logger.Logger) error {
	if err := mapstructure.Decode(config, p); err != nil {
		return err
	}

	p.alias = alias
	p.pipe = pipeline
	p.log = log

	if len(p.Code) == 0 && len(p.File) == 0 {
		return errors.New("code or file required")
	}

	if len(p.Code) > 0 && len(p.File) > 0 {
		return errors.New("both code or file cannot be set")
	}

	p.thread = &starlark.Thread{
		Print: func(_ *starlark.Thread, msg string) {
			p.log.Debugf("from starlark: %v", msg)
		},
	}

	builtins := starlark.StringDict{}
	builtins["newEvent"] = starlark.NewBuiltin("newEvent", NewEvent)

	var src any
	if len(p.Code) > 0 {
		src = p.Code
	}
	_, program, err := starlark.SourceProgram(p.File, src, builtins.Has)
	if err != nil {
		return fmt.Errorf("compilation error: %v", err)
	}

	globals, err := program.Init(p.thread, builtins)
	if err != nil {
		return fmt.Errorf("initialization error: %v", err)
	}

	stFunc, ok := globals["process"]
	if !ok {
		return errors.New("process(event) function not found in starlark program")
	}
	p.stFunc = stFunc

	return nil
}

func (p *Starlark) Prepare(
	in <-chan *core.Event,
	out chan<- *core.Event,
) {
	p.in = in
	p.out = out
}

func (p *Starlark) Close() error {
	return nil
}

func (p *Starlark) Alias() string {
	return p.alias
}

func (p *Starlark) Run() {
	for e := range p.in {
		now := time.Now()
		result, err := starlark.Call(p.thread, p.stFunc, []starlark.Value{&event{event: e}}, nil)
		if err != nil {
			e.StackError(err)
			p.out <- e
			metrics.ObserveProcessorSummary("starlark", p.alias, p.pipe, metrics.EventFailed, time.Since(now))
			continue
		}

		events, err := unpackEvents(result)
		if err != nil {
			e.StackError(err)
			p.out <- e
			metrics.ObserveProcessorSummary("starlark", p.alias, p.pipe, metrics.EventFailed, time.Since(now))
			continue
		}

		for _, event := range events {
			p.out <- event
		}
		metrics.ObserveProcessorSummary("starlark", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
	}
}

func unpackEvents(starValue starlark.Value) ([]*core.Event, error) {
	events := []*core.Event{}
	switch v := starValue.(type) {
	case *event:
		events = append(events, v.event)
	case *starlark.List:
		iter := v.Iterate()
		var value starlark.Value
		for iter.Next(&value) {
			r, err := unpackEvents(value)
			if err != nil {
				return nil, err
			}
			events = append(events, r...)
		}
	case *starlark.NoneType:
		return events, nil
	}
	return events, nil
}

func init() {
	plugins.AddProcessor("starlark", func() core.Processor {
		return &Starlark{}
	})
}
