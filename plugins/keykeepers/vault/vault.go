package vault

import (
	"regexp"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
)

var secretKeyPattern = regexp.MustCompile(`([a-zA-Z_-/]+)#([a-zA-Z_-\.]+)`)

type Vault struct {
	*core.BaseKeykeeper `mapstructure:"-"`
	Address   string `mapstructure:"address"`
	MountPath string `mapstructure:"mount_path"`
	PathPrefix string `mapstructure:"path_prefix"` // e.g. dev/, test/, prod/
	KvVersion string `mapstructure:"kv_version"` // v1, v2

	Auth Auth `mapstructure:"auth"`
}

type Auth struct {
	Method string `mapstructure:"method"` // k8s, approle

}

func (k *Vault) Init() error {
	return nil
}

func (k *Vault) Get(key string) (any, error) {
	return nil, nil
}

func (k *Vault) Close() error {
	return nil
}

func init() {
	plugins.AddKeykeeper("vault", func() core.Keykeeper {
		return &Vault{}
	})
}
