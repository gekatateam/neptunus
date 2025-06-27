package exec

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/convert"
	"github.com/gekatateam/neptunus/plugins/common/elog"
)

type Exec struct {
	*core.BaseOutput `mapstructure:"-"`
	Command          string            `mapstructure:"command"`
	Timeout          time.Duration     `mapstructure:"timeout"`
	Args             []string          `mapstructure:"args"`
	EnvLabels        map[string]string `mapstructure:"envlabels"`
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
				elog.EventGroup(e),
			)
			o.Done <- e
			o.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		args, err := o.unpackArgs(e)
		if err != nil {
			o.Log.Error("command args preparation failed",
				"error", err,
				elog.EventGroup(e),
			)
			o.Done <- e
			o.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), o.Timeout)

		cmd := exec.CommandContext(ctx, o.Command, args...)
		cmd.Env = append(cmd.Env, os.Environ()...)
		cmd.Env = append(cmd.Env, envs...)

		if err := cmd.Run(); err != nil {
			o.Log.Error("command execution failed",
				"error", err,
				elog.EventGroup(e),
			)
			o.Observe(metrics.EventFailed, time.Since(now))
		} else {
			o.Log.Debug("command execution completed",
				elog.EventGroup(e),
			)
			o.Observe(metrics.EventAccepted, time.Since(now))
		}

		o.Done <- e
		cancel()
	}
}

func (o *Exec) Close() error {
	return nil
}

func (p *Exec) unpackEnvs(e *core.Event) ([]string, error) {
	var envs []string

	for k, v := range p.EnvLabels {
		label, ok := e.GetLabel(v)
		if !ok {
			return nil, fmt.Errorf("no such label: %v", v)
		}

		envs = append(envs, k+"="+label)
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
