package env

import (
	"os"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
)

type Fs struct {
	*core.BaseKeykeeper `mapstructure:"-"`
}

func (k *Fs) Init() error {
	return nil
}

func (k *Fs) Get(path string) (any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return string(content), nil
}

func (k *Fs) Close() error {
	return nil
}

func init() {
	plugins.AddKeykeeper("fs", func() core.Keykeeper {
		return &Fs{}
	})
}
