package self

import (
	"github.com/gekatateam/mappath"
	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
)

type Self struct {
	*core.BaseKeykeeper `mapstructure:"-"`
	cfg                 map[string]any
}

func (k *Self) Init() error {
	return nil
}

func (k *Self) SetConfig(config *config.Pipeline) {
	k.cfg = make(map[string]any)
	k.cfg["vars"] = config.Vars
	k.cfg["settings"] = map[string]any{
		"id":          config.Settings.Id,
		"lines":       config.Settings.Lines,
		"run":         config.Settings.Run,
		"buffer":      config.Settings.Buffer,
		"consistency": config.Settings.Consistency,
	}
}

func (k *Self) Get(key string) (any, error) {
	return mappath.Get(k.cfg, key)
}

func (k *Self) Close() error {
	return nil
}

func init() {
	plugins.AddKeykeeper("self", func() core.Keykeeper {
		return &Self{}
	})
}
