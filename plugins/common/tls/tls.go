package tls

import (
	"crypto/tls"
	"fmt"

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

	cfg, err := pkg.NewConfigBuilder().
		RootCaFile(t.CAFile).
		KeyPairFile(t.CertFile, t.KeyFile).
		SkipVerify(t.InsecureSkipVerify).
		ServerName(t.ServerName).
		Build()
	if err != nil {
		return nil, fmt.Errorf("tls: %v", err)
	}

	return cfg, nil
}

type TLSServerConfig struct {
	KeyFile        string   `mapstructure:"tls_key_file"`
	CertFile       string   `mapstructure:"tls_cert_file"`
	AllovedCACerts []string `mapstructure:"tls_allowed_cacerts"`
	MinVersion     string   `mapstructure:"tls_min_version"`
	MaxVersion     string   `mapstructure:"tls_max_version"`
	Enable         bool     `mapstructure:"tls_enable"`
}

func (t *TLSServerConfig) Config() (*tls.Config, error) {
	if !t.Enable {
		return nil, nil
	}

	builder := pkg.NewConfigBuilder().
		KeyPairFile(t.CertFile, t.KeyFile).
		MinMaxVersion(t.MinVersion, t.MaxVersion)

	for _, v := range t.AllovedCACerts {
		builder = builder.ClientCaFile(v)
	}

	cfg, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("tls: %v", err)
	}

	return cfg, nil
}
