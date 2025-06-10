package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

type tlsConfigBuilder struct {
	cfg *tls.Config
	err error
}

func NewConfigBuilder() *tlsConfigBuilder {
	return &tlsConfigBuilder{
		cfg: &tls.Config{},
	}
}

func (tb *tlsConfigBuilder) SkipVerify(skip bool) *tlsConfigBuilder {
	if tb.err != nil {
		return tb
	}
	tb.cfg.InsecureSkipVerify = skip
	return tb
}

func (tb *tlsConfigBuilder) KeyPairFile(cert, key string) *tlsConfigBuilder {
	if tb.err != nil {
		return tb
	}
	if len(cert) > 0 && len(key) > 0 {
		cert, err := os.ReadFile(cert)
		if err != nil {
			tb.err = err
			return tb
		}
		key, err := os.ReadFile(key)
		if err != nil {
			tb.err = err
			return tb
		}
		return tb.KeyPairRaw(cert, key)
	}
	return tb
}

func (tb *tlsConfigBuilder) KeyPairRaw(cert, key []byte) *tlsConfigBuilder {
	if tb.err != nil {
		return tb
	}

	if tb.cfg.Certificates == nil {
		tb.cfg.Certificates = []tls.Certificate{}
	}

	if len(cert) > 0 && len(key) > 0 {
		c, err := tls.X509KeyPair(cert, key)
		if err != nil {
			tb.err = err
			return tb
		}
		tb.cfg.Certificates = append(tb.cfg.Certificates, c)
	}
	return tb
}

func (tb *tlsConfigBuilder) RootCaFile(ca string) *tlsConfigBuilder {
	if tb.err != nil {
		return tb
	}
	if len(ca) > 0 {
		caCert, err := os.ReadFile(ca)
		if err != nil {
			tb.err = err
			return tb
		}
		return tb.RootCaRaw(caCert)
	}
	return tb
}

func (tb *tlsConfigBuilder) RootCaRaw(ca []byte) *tlsConfigBuilder {
	if tb.err != nil {
		return tb
	}

	if tb.cfg.RootCAs == nil {
		tb.cfg.RootCAs = x509.NewCertPool()
	}

	tb.cfg.RootCAs.AppendCertsFromPEM(ca)
	return tb
}

func (tb *tlsConfigBuilder) ClientCaFile(ca string) *tlsConfigBuilder {
	if tb.err != nil {
		return tb
	}
	if len(ca) > 0 {
		caCert, err := os.ReadFile(ca)
		if err != nil {
			tb.err = err
			return tb
		}
		return tb.ClientCaRaw(caCert)
	}
	return tb
}

func (tb *tlsConfigBuilder) ClientCaRaw(ca []byte) *tlsConfigBuilder {
	if tb.err != nil {
		return tb
	}

	if tb.cfg.ClientCAs == nil {
		tb.cfg.ClientCAs = x509.NewCertPool()
	}

	tb.cfg.ClientCAs.AppendCertsFromPEM(ca)
	tb.cfg.ClientAuth = tls.RequireAndVerifyClientCert
	return tb
}

func (tb *tlsConfigBuilder) ServerName(serverName string) *tlsConfigBuilder {
	if tb.err != nil {
		return tb
	}
	tb.cfg.ServerName = serverName
	return tb
}

func (tb *tlsConfigBuilder) MinMaxVersion(min, max string) *tlsConfigBuilder {
	if tb.err != nil {
		return tb
	}

	if len(min) > 0 {
		if minVersion, ok := TlsVersion[min]; !ok {
			tb.err = fmt.Errorf("unknown TLS version: %v", min)
			return tb
		} else {
			tb.cfg.MinVersion = minVersion
		}
	}

	if len(max) > 0 {
		if maxVersion, ok := TlsVersion[max]; !ok {
			tb.err = fmt.Errorf("unknown TLS version: %v", min)
			return tb
		} else {
			tb.cfg.MaxVersion = maxVersion
		}
	}

	return tb
}

func (tb *tlsConfigBuilder) Reset() *tlsConfigBuilder {
	tb.cfg = &tls.Config{}
	tb.err = nil
	return tb
}

func (tb *tlsConfigBuilder) Build() (*tls.Config, error) {
	return tb.cfg, tb.err
}
