package exec

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/convert"
)

type Exec struct {
	*core.BaseOutput `mapstructure:"-"`
	Command          string        `mapstructure:"command"`
	Timeout          time.Duration `mapstructure:"timeout"`
	Envs             []string      `mapstructure:"envs"`
	Args             []string      `mapstructure:"args"`
}

func (o *Exec) Init() error {
	if len(o.Command) == 0 {
		return errors.New("command required")
	}

	return nil
}

func (o *Exec) Run() {
	for e := range o.In {
		now := time.Now()

		envs, err := o.unpackEnvs(e)
		if err != nil {
			o.Log.Error("environment variables preparation failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			o.Done <- e
			o.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		args, err := o.unpackArgs(e)
		if err != nil {
			o.Log.Error("command args preparation failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			o.Done <- e
			o.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), o.Timeout)

		cmd := exec.CommandContext(ctx, o.Command, args...)
		cmd.Env = append(cmd.Env, os.Environ()...)
		cmd.Env = append(cmd.Env, envs...)

		o.Done <- e
		if err := cmd.Run(); err != nil {
			o.Log.Error("command execution failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			o.Observe(metrics.EventFailed, time.Since(now))
		} else {
			o.Log.Debug("command execution completed",
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			o.Observe(metrics.EventAccepted, time.Since(now))
		}

		cancel()
	}
}

func (o *Exec) Close() error {
	return nil
}

func (o *Exec) unpackEnvs(e *core.Event) ([]string, error) {
	var envs []string

	for _, v := range o.Envs {
		label, ok := e.GetLabel(v)
		if !ok {
			return nil, fmt.Errorf("no such label: %v", v)
		}

		envs = append(envs, v+"="+label)
	}

	return envs, nil
}

func (o *Exec) unpackArgs(e *core.Event) ([]string, error) {
	var args []string

	for _, v := range o.Args {
		field, err := e.GetField(v)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", v, err)
		}

		switch f := field.(type) {
		case []any:
			for i, j := range f {
				arg, err := convert.AnyToString(j)
				if err != nil {
					return nil, fmt.Errorf("%v.%v: %w", v, i, err)
				}

				args = append(args, arg)
			}
		case map[string]any:
			for k, j := range f {
				arg, err := convert.AnyToString(j)
				if err != nil {
					return nil, fmt.Errorf("%v.%v: %w", v, k, err)
				}

				args = append(args, k, arg)
			}
		default:
			arg, err := convert.AnyToString(f)
			if err != nil {
				return nil, fmt.Errorf("%v: %w", v, err)
			}

			args = append(args, arg)
		}
	}

	return args, nil
}

func init() {
	plugins.AddOutput("exec", func() core.Output {
		return &Exec{
			Timeout: 10 * time.Second,
		}
	})
}
