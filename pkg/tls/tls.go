package tls

import (
	"crypto/tls"
	"crypto/x509"
	"os"
)

type tlsConfigBuilder struct {
	cfg *tls.Config
	err error
}

func NewConfigBuilder() *tlsConfigBuilder {
	return &tlsConfigBuilder{
		cfg: &tls.Config{
			RootCAs:      x509.NewCertPool(),
			ClientCAs:    x509.NewCertPool(),
			Certificates: []tls.Certificate{},
		},
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
	if len(cert) > 0 && len(key) > 0 {
		c, err := tls.X509KeyPair(cert, key)
		if err != nil {
			tb.err = err
			return tb
		}
		tb.cfg.Certificates = append(tb.cfg.Certificates, c)
		//tb.cfg.BuildNameToCertificate() // deprecated in 1.14
	}
	return tb
}

func (tb *tlsConfigBuilder) CaFile(ca string) *tlsConfigBuilder {
	if tb.err != nil {
		return tb
	}
	if len(ca) > 0 {
		caCert, err := os.ReadFile(ca)
		if err != nil {
			tb.err = err
			return tb
		}
		return tb.CaRaw(caCert)
	}
	return tb
}

func (tb *tlsConfigBuilder) CaRaw(ca []byte) *tlsConfigBuilder {
	if tb.err != nil {
		return tb
	}
	tb.cfg.RootCAs.AppendCertsFromPEM(ca)
	return tb
}

func (tb *tlsConfigBuilder) ServerName(serverName string) *tlsConfigBuilder {
	if tb.err != nil {
		return tb
	}
	tb.cfg.ServerName = serverName
	return tb
}

func (tb *tlsConfigBuilder) Build() (*tls.Config, error) {
	return tb.cfg, tb.err
}
