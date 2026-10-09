package ollama

import (
	"errors"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
)

type Ollama struct {
	*core.BaseProcessor `mapstructure:"-"`
}

func (p *Ollama) Init() error {
	panic(errors.ErrUnsupported)
}

func (p *Ollama) Close() error {
	panic(errors.ErrUnsupported)
}

func (p *Ollama) Run() {
	panic(errors.ErrUnsupported)
}

func init() {
	plugins.AddProcessor("ollama", func() core.Processor {
		return &Ollama{}
	})
}
