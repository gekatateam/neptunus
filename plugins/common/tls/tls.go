package tls

import (
	"crypto/tls"

	pkg "github.com/gekatateam/neptunus/pkg/tls"
)

type TLSClientConfig struct {
	KeyFile            string `mapstructure:"tls_key_file"`
	CertFile           string `mapstructure:"tls_cert_file"`
	CAFile             string `mapstructure:"tls_ca_file"`
	ServerName         string `mapstructure:"tls_server_name"`
	InsecureSkipVerify bool   `mapstructure:"tls_insecure_skip_verify"`
	Enable             bool   `mapstructure:"tls_enable"`
}

func (t *TLSClientConfig) Config() (*tls.Config, error) {
	if !t.Enable {
		return nil, nil
	}

	return pkg.NewConfigBuilder().
		CaFile(t.CAFile).
		KeyPairFile(t.CAFile, t.KeyFile).
		SkipVerify(t.InsecureSkipVerify).
		ServerName(t.ServerName).
		Build()
}
