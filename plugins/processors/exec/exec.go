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
	*core.BaseProcessor `mapstructure:"-"`
	Command             string        `mapstructure:"command"`
	Timeout             time.Duration `mapstructure:"timeout"`
	Envs                []string      `mapstructure:"envs"`
	Args                []string      `mapstructure:"args"`
	ExecCodeTo          string        `mapstructure:"exec_code_to"`
	ExecOutputTo        string        `mapstructure:"exec_output_to"`
}

func (p *Exec) Init() error {
	if len(p.Command) == 0 {
		return errors.New("command required")
	}

	return nil
}

func (p *Exec) Run() {
	for e := range p.In {
		now := time.Now()

		envs, err := p.unpackEnvs(e)
		if err != nil {
			p.Log.Error("environment variables preparation failed",
				"error", err,
				elog.EventGroup(e),
			)
			e.StackError(err)
			p.Out <- e
			p.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		args, err := p.unpackArgs(e)
		if err != nil {
			p.Log.Error("command args preparation failed",
				"error", err,
				elog.EventGroup(e),
			)
			e.StackError(err)
			p.Out <- e
			p.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
		cmd := exec.CommandContext(ctx, p.Command, args...)
		cmd.Env = append(cmd.Env, os.Environ()...)
		cmd.Env = append(cmd.Env, envs...)

		out, err := cmd.CombinedOutput()
		var exitErr *exec.ExitError
		if err != nil && !errors.As(err, &exitErr) {
			p.Log.Error("command execution failed",
				"error", err,
				elog.EventGroup(e),
			)
			e.StackError(err)
			p.Out <- e
			p.Observe(metrics.EventFailed, time.Since(now))
		}

		exitCode := 0
		if err != nil && errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}

		if err := e.SetField(p.ExecCodeTo, exitCode); err != nil {
			p.Log.Warn(fmt.Sprintf("set field %v failed", p.ExecCodeTo),
				"error", err,
			)
		}
		if err := e.SetField(p.ExecOutputTo, string(out)); err != nil {
			p.Log.Warn(fmt.Sprintf("set field %v failed", p.ExecOutputTo),
				"error", err,
			)
		}

		p.Log.Debug("command execution completed",
			elog.EventGroup(e),
		)
		p.Out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))

		cancel()
	}
}

func (p *Exec) Close() error {
	return nil
}

func (p *Exec) unpackEnvs(e *core.Event) ([]string, error) {
	var envs []string

	for _, v := range p.Envs {
		label, ok := e.GetLabel(v)
		if !ok {
			return nil, fmt.Errorf("no such label: %v", v)
		}

		envs = append(envs, v+"="+label)
	}

	return envs, nil
}

func (p *Exec) unpackArgs(e *core.Event) ([]string, error) {
	var args []string

	for _, v := range p.Args {
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
	plugins.AddProcessor("exec", func() core.Processor {
		return &Exec{
			Timeout:      10 * time.Second,
			ExecCodeTo:   "exec.code",
			ExecOutputTo: "exec.output",
		}
	})
}
