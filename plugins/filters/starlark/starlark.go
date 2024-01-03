package starlark

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.starlark.net/starlark"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
	common "github.com/gekatateam/neptunus/plugins/common/starlark"
)

type Starlark struct {
	alias            string
	pipe             string
	*common.Starlark `mapstructure:",squash"`

	stThread *starlark.Thread
	stFunc   *starlark.Function

	in       <-chan *core.Event
	rejected chan<- *core.Event
	accepted chan<- *core.Event
	log      *slog.Logger
}

func (f *Starlark) Init(config map[string]any, alias, pipeline string, log *slog.Logger) error {
	if err := mapstructure.Decode(config, f); err != nil {
		return err
	}

	if err := f.Starlark.Init(alias, log); err != nil {
		return err
	}

	f.alias = alias
	f.pipe = pipeline
	f.log = log

	stFunc, err := f.Starlark.Func("filter")
	if err != nil {
		return err
	}
	f.stFunc = stFunc
	f.stThread = f.Starlark.Thread()

	return nil
}

func (f *Starlark) SetChannels(
	in <-chan *core.Event,
	rejected chan<- *core.Event,
	accepted chan<- *core.Event,
) {
	f.in = in
	f.rejected = rejected
	f.accepted = accepted
}

func (f *Starlark) Close() error {
	return nil
}

func (f *Starlark) Run() {
	for e := range f.in {
		now := time.Now()
		result, err := starlark.Call(f.stThread, f.stFunc, []starlark.Value{common.ROEvent(e)}, nil)
		if err != nil {
			f.log.Error("exec failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.StackError(fmt.Errorf("exec failed: %v", err))
			e.AddTag("::starlark_filtering_failed")
			f.rejected <- e
			metrics.ObserveFliterSummary("starlark", f.alias, f.pipe, metrics.EventRejected, time.Since(now))
			continue
		}

		ok, err := unpack(result)
		if err != nil {
			f.log.Error("exec failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.StackError(fmt.Errorf("exec failed: %v", err))
			e.AddTag("::starlark_filtering_failed")
			f.rejected <- e
			metrics.ObserveFliterSummary("starlark", f.alias, f.pipe, metrics.EventRejected, time.Since(now))
			continue
		}

		if ok {
			f.accepted <- e
			metrics.ObserveFliterSummary("starlark", f.alias, f.pipe, metrics.EventAccepted, time.Since(now))
		} else {
			f.rejected <- e
			metrics.ObserveFliterSummary("starlark", f.alias, f.pipe, metrics.EventRejected, time.Since(now))
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
