package env

import (
	"fmt"
	"os"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
)

type Env struct {
	*core.BaseKeykeeper `mapstructure:"-"`
}

func (k *Env) Init() error {
	return nil
}

func (k *Env) Get(key string) (any, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return nil, fmt.Errorf("no such env: %v", key)
	}

	return val, nil
}

func (k *Env) Close() error {
	return nil
}

func init() {
	plugins.AddKeykeeper("env", func() core.Keykeeper {
		return &Env{}
	})
}
