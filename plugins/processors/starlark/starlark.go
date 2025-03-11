package starlark

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.starlark.net/starlark"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	common "github.com/gekatateam/neptunus/plugins/common/starlark"
)

type Starlark struct {
	*core.BaseProcessor `mapstructure:"-"`
	*common.Starlark    `mapstructure:",squash"`

	stThread *starlark.Thread
	stFunc   *starlark.Function
}

func (p *Starlark) Init() error {
	if err := p.Starlark.Init(p.Alias, p.Log); err != nil {
		return err
	}

	stFunc, err := p.Starlark.Func("process")
	if err != nil {
		return err
	}
	p.stFunc = stFunc
	p.stThread = p.Starlark.Thread()

	return nil
}

func (p *Starlark) Close() error {
	return nil
}

func (p *Starlark) Run() {
	for e := range p.In {
		now := time.Now()
		result, err := starlark.Call(p.stThread, p.stFunc, []starlark.Value{common.RWEvent(e)}, nil)
		if err != nil {
			p.Log.Error("exec failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.StackError(fmt.Errorf("exec failed: %v", err))
			e.AddTag("::starlark_processing_failed")
			p.Out <- e
			p.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		events, err := unpack(result)
		if err != nil {
			p.Log.Error("exec failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.StackError(fmt.Errorf("exec failed: %v", err))
			e.AddTag("::starlark_processing_failed")
			p.Out <- e
			p.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		p.markAsDone(e, events)
		for _, event := range events {
			p.Out <- event
		}
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

// check if origin event in returned list
// if it's not, drop it
func (p *Starlark) markAsDone(e *core.Event, events []*core.Event) {
	for _, v := range events {
		if v.UUID == e.UUID {
			return
		}
	}
	p.Drop <- e
}

func unpack(starValue starlark.Value) ([]*core.Event, error) {
	events := []*core.Event{}

	switch v := starValue.(type) {
	case *common.Event:
		return append(events, v.Event()), nil
	case common.Error:
		return nil, errors.New(v.String())
	case *starlark.List:
		iter := v.Iterate()
		defer iter.Done()
		var value starlark.Value
		for iter.Next(&value) {
			r, err := unpack(value)
			if err != nil {
				return nil, err
			}
			events = append(events, r...)
		}
		return events, nil
	case *starlark.NoneType:
		return events, nil
	}

	return nil, fmt.Errorf("unknown function result, expected event, events list, error or none, got %v", starValue.Type())
}

func init() {
	plugins.AddProcessor("starlark", func() core.Processor {
		return &Starlark{
			Starlark: &common.Starlark{},
		}
	})
}
