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
	*core.BaseFilter `mapstructure:"-"`
	*common.Starlark `mapstructure:",squash"`

	stThread *starlark.Thread
	stFunc   *starlark.Function
}

func (f *Starlark) Init() error {
	if err := f.Starlark.Init(f.Alias, f.Log); err != nil {
		return err
	}

	stFunc, err := f.Starlark.Func("filter")
	if err != nil {
		return err
	}
	f.stFunc = stFunc
	f.stThread = f.Starlark.Thread()

	return nil
}

func (f *Starlark) Close() error {
	return nil
}

func (f *Starlark) Run() {
	for e := range f.In {
		now := time.Now()
		result, err := starlark.Call(f.stThread, f.stFunc, []starlark.Value{common.ROEvent(e)}, nil)
		if err != nil {
			f.Log.Error("exec failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.StackError(fmt.Errorf("exec failed: %v", err))
			e.AddTag("::starlark_filtering_failed")
			f.Rej <- e
			f.Observe(metrics.EventRejected, time.Since(now))
			continue
		}

		ok, err := unpack(result)
		if err != nil {
			f.Log.Error("exec failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.StackError(fmt.Errorf("exec failed: %v", err))
			e.AddTag("::starlark_filtering_failed")
			f.Rej <- e
			f.Observe(metrics.EventRejected, time.Since(now))
			continue
		}

		if ok {
			f.Acc <- e
			f.Observe(metrics.EventAccepted, time.Since(now))
		} else {
			f.Rej <- e
			f.Observe(metrics.EventRejected, time.Since(now))
		}
	}
}

func unpack(starValue starlark.Value) (bool, error) {
	switch v := starValue.(type) {
	case common.Error:
		return false, errors.New(v.String())
	case starlark.Bool:
		return bool(v), nil
	}

	return false, fmt.Errorf("unknown function result, expected error or bool, got %v", starValue.Type())
}

func init() {
	plugins.AddFilter("starlark", func() core.Filter {
		return &Starlark{
			Starlark: &common.Starlark{},
		}
	})
}
