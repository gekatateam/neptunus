package exec

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/core/lookup"
)

type Exec struct {
	*core.BaseLookup `mapstructure:"-"`
	Command          string            `mapstructure:"command"`
	Timeout          time.Duration     `mapstructure:"timeout"`
	Args             []string          `mapstructure:"args"`
	Envs             map[string]string `mapstructure:"envs"`

	parser core.Parser
	envs   []string
}

func (l *Exec) Init() error {
	if len(l.Command) == 0 {
		return errors.New("command required")
	}

	l.envs = append(l.envs, os.Environ()...)
	for k, v := range l.Envs {
		l.envs = append(l.envs, k+"="+v)
	}

	return nil
}

func (l *Exec) Close() error {
	return l.parser.Close()
}

func (l *Exec) SetParser(p core.Parser) {
	l.parser = p
}

func (l *Exec) Update() (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), l.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, l.Command, l.Args...)
	cmd.Env = l.envs

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	event, err := l.parser.Parse(out, "")
	if err != nil {
		return nil, err
	}

	if len(event) == 0 {
		return nil, errors.New("parser returns zero events, nothing to store")
	}

	if len(event) > 1 {
		l.Log.Warn("parser returns more than one event, only first event will be used for lookup data")
	}

	return event[0].Data, nil
}

func init() {
	plugins.AddLookup("exec", func() core.Lookup {
		return &lookup.Lookup{
			LazyLookup: &Exec{
				Timeout: 10 * time.Second,
			},
			Interval: 30 * time.Second,
		}
	})
}
