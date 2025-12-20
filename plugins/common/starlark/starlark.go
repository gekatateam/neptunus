package starlark

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"go.starlark.net/lib/json"
	"go.starlark.net/lib/math"
	starlarktime "go.starlark.net/lib/time"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"

	"github.com/qri-io/starlib/encoding/base64"
	"github.com/qri-io/starlib/encoding/csv"
	"github.com/qri-io/starlib/encoding/yaml"
	"github.com/qri-io/starlib/re"

	"github.com/gekatateam/neptunus/pkg/starlarkdate"
	"github.com/gekatateam/neptunus/pkg/starlarkfs"
)

type Starlark struct {
	alias  string
	Code   string         `mapstructure:"code"`
	File   string         `mapstructure:"file"`
	Consts map[string]any `mapstructure:"constants"`

	thread  *starlark.Thread
	globals starlark.StringDict

	log *slog.Logger
}

func (p *Starlark) Init(alias string, log *slog.Logger) error {
	p.alias = alias
	p.log = log

	if len(p.Code) == 0 && len(p.File) == 0 {
		return errors.New("code or file required")
	}

	if len(p.Code) > 0 {
		p.File = p.alias + ".star"
		goto SCRIPT_LOADED
	}

	if len(p.File) > 0 {
		script, err := os.ReadFile(p.File)
		if err != nil {
			return fmt.Errorf("failed to load %v script: %w", p.File, err)
		}
		p.Code = string(script)
	}
SCRIPT_LOADED:

	builtins := starlark.StringDict{
		"newEvent": starlark.NewBuiltin("newEvent", newEvent),
		"error":    starlark.NewBuiltin("error", newError),
		"struct":   starlark.NewBuiltin("struct", starlarkstruct.Make),
		"handle":   starlark.NewBuiltin("handle", handle),
	}

	if len(p.Consts) > 0 {
		for k, v := range p.Consts {
			sv, err := ToStarlarkValue(v)
			if err != nil {
				return fmt.Errorf("constants: %v: %w", k, err)
			}

			builtins[k] = sv
		}
	}

	opts := &syntax.FileOptions{
		Set:             false,
		While:           true,
		TopLevelControl: true,
		GlobalReassign:  true,
	}

	p.thread = &starlark.Thread{
		Print: func(_ *starlark.Thread, msg string) {
			p.log.Debug(fmt.Sprintf("from starlark: %v", msg))
		},
		Load: func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
			switch module {
			case "log.star":
				return starlark.StringDict{
					"log": Log,
				}, nil
			case "fs.star":
				return starlark.StringDict{
					"fs": starlarkfs.Module,
				}, nil
			case "math.star":
				return starlark.StringDict{
					"math": math.Module,
				}, nil
			case "time.star":
				return starlark.StringDict{
					"time": starlarktime.Module,
				}, nil
			case "date.star":
				return starlark.StringDict{
					"date": starlarkdate.Module,
				}, nil
			case "json.star":
				return starlark.StringDict{
					"json": json.Module,
				}, nil
			case "yaml.star":
				return yaml.LoadModule()
			case "csv.star":
				return csv.LoadModule()
			case "base64.star":
				return base64.LoadModule()
			case "re.star":
				return re.LoadModule()
			default:
				script, err := os.ReadFile(module)
				if err != nil {
					return nil, err
				}

				entries, err := starlark.ExecFileOptions(opts, thread, module, script, builtins)
				if err != nil {
					return nil, err
				}

				return entries, nil
			}
		},
	}

	p.thread.SetLocal(loggerKey, log)

	_, program, err := starlark.SourceProgramOptions(opts, p.File, p.Code, builtins.Has)
	if err != nil {
		return fmt.Errorf("compilation failed: %w", err)
	}

	globals, err := program.Init(p.thread, builtins)
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}
	p.globals = globals

	return nil
}

func (p *Starlark) Thread() *starlark.Thread {
	return p.thread
}

func (p *Starlark) Func(name string) (*starlark.Function, error) {
	stVal, ok := p.globals[name]
	if !ok {
		return nil, fmt.Errorf("%v function not found in starlark script", name)
	}

	stFunc, ok := stVal.(*starlark.Function)
	if !ok {
		return nil, fmt.Errorf("%v is not a function", name)
	}

	return stFunc, nil
}
