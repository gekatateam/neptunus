package exec

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/convert"
	"github.com/gekatateam/neptunus/plugins/common/elog"
)

type Exec struct {
	*core.BaseProcessor `mapstructure:"-"`
	Command             string            `mapstructure:"command"`
	Timeout             time.Duration     `mapstructure:"timeout"`
	Args                []string          `mapstructure:"args"`
	StdinField          string            `mapstructure:"stdin_field"`
	ExecCodeTo          string            `mapstructure:"exec_code_to"`
	ExecOutputTo        string            `mapstructure:"exec_output_to"`
	Envs                map[string]string `mapstructure:"envs"`
	EnvLabels           map[string]string `mapstructure:"envlabels"`

	envs []string
}

func (p *Exec) Init() error {
	if len(p.Command) == 0 {
		return errors.New("command required")
	}

	p.envs = append(p.envs, os.Environ()...)
	for k, v := range p.Envs {
		p.envs = append(p.envs, k+"="+v)
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

		stdin, err := p.unpackStdin(e)
		if err != nil {
			p.Log.Error("command stdin preparation failed",
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
		cmd.Env = slices.Concat(p.envs, envs)

		if len(stdin) > 0 {
			cmd.Stdin = strings.NewReader(stdin)
		}

		out, err := cmd.CombinedOutput()
		cancel()

		exitCode := 0
		if err != nil {
			if exitError, ok := errors.AsType[*exec.ExitError](err); ok {
				exitCode = exitError.ExitCode()
			} else { // any other error means that command execution failed
				p.Log.Error("command execution failed",
					"error", fmt.Errorf("%w: %v", err, string(out)),
					elog.EventGroup(e),
				)
				e.StackError(err)
				p.Out <- e
				p.Observe(metrics.EventFailed, time.Since(now))
				continue
			}
		}

		if err := e.SetField(p.ExecCodeTo, exitCode); err != nil {
			p.Log.Error(fmt.Sprintf("set field %v failed", p.ExecCodeTo),
				"error", err,
				elog.EventGroup(e),
			)
			e.StackError(err)
			p.Out <- e
			p.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		if err := e.SetField(p.ExecOutputTo, string(out)); err != nil {
			p.Log.Error(fmt.Sprintf("set field %v failed", p.ExecOutputTo),
				"error", err,
				elog.EventGroup(e),
			)
			e.StackError(err)
			p.Out <- e
			p.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		p.Log.Debug("command execution completed",
			elog.EventGroup(e),
		)
		p.Out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func (p *Exec) Close() error {
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

func (p *Exec) unpackStdin(e *core.Event) (string, error) {
	if len(p.StdinField) == 0 {
		return "", nil
	}

	field, err := e.GetField(p.StdinField)
	if err != nil {
		return "", fmt.Errorf("%v: %w", p.StdinField, err)
	}

	stdin, err := convert.AnyToString(field)
	if err != nil {
		return "", fmt.Errorf("%v: %w", p.StdinField, err)
	}

	return stdin, nil
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
