package vault

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"

	vault "github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	pkgtls "github.com/gekatateam/neptunus/plugins/common/tls"
)

var secretKeyPattern = regexp.MustCompile(`([a-zA-Z_\-/]+)#([a-zA-Z_\-\.]+)`)

type Vault struct {
	*core.BaseKeykeeper `mapstructure:"-"`
	Address             string  `mapstructure:"address"`
	MountPath           string  `mapstructure:"mount_path"`
	PathPrefix          string  `mapstructure:"path_prefix"` // e.g. dev/, test/, prod/
	KvVersion           string  `mapstructure:"kv_version"`  // v1, v2
	Namespace           string  `mapstructure:"namespace"`
	Approle             Approle `mapstructure:"approle"`
	K8s                 K8s     `mapstructure:"k8s"`

	*pkgtls.TLSClientConfig `mapstructure:",squash"`

	client     *vault.Client
	secretFunc func(path, secret string) (any, error)
}

type Approle struct {
	MountPath string `mapstructure:"mount_path"`
	RoleId    string `mapstructure:"role_id"`
	SecretId  string `mapstructure:"secret_id"`
}

type K8s struct {
	MountPath string `mapstructure:"mount_path"`
	RoleName  string `mapstructure:"role_name"`
	TokenPath string `mapstructure:"token_path"`
}

func (k *Vault) Init() error {
	if len(k.Address) == 0 {
		return errors.New("address required")
	}

	if len(k.MountPath) == 0 {
		return errors.New("mount_path required")
	}

	switch k.KvVersion {
	case "v1":
		k.secretFunc = k.readV1
	case "v2":
		k.secretFunc = k.readV2
	default:
		return fmt.Errorf("unknown kv engine version: %v", k.KvVersion)
	}

	tlscfg, err := k.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	cfg := vault.DefaultConfiguration()
	httpTransport := cfg.HTTPClient.Transport.(*http.Transport)
	httpTransport.TLSClientConfig = tlscfg

	client, err := vault.New(
		vault.WithConfiguration(cfg),
		vault.WithAddress(k.Address),
	)
	if err != nil {
		return err
	}

	k.client = client

	if len(k.Approle.RoleId) > 0 && len(k.Approle.SecretId) > 0 {
		resp, err := k.client.Auth.AppRoleLogin(context.Background(), schema.AppRoleLoginRequest{
			RoleId:   k.Approle.RoleId,
			SecretId: k.Approle.SecretId,
		}, vault.WithMountPath(k.Approle.MountPath))
		if err != nil {
			return fmt.Errorf("approle authentication failed: %w", err)
		}

		if err := k.client.SetToken(resp.Auth.ClientToken); err != nil {
			return fmt.Errorf("token set failed: %w", err)
		}

		goto CLIENT_AUTH_SUCCESS
	}

	if len(k.K8s.RoleName) > 0 && len(k.K8s.TokenPath) > 0 {
		jwt, err := os.ReadFile(k.K8s.TokenPath)
		if err != nil {
			return fmt.Errorf("unable to read file containing service account token: %w", err)
		}

		resp, err := k.client.Auth.KubernetesLogin(context.Background(), schema.KubernetesLoginRequest{
			Jwt:  string(jwt),
			Role: k.K8s.RoleName,
		}, vault.WithMountPath(k.K8s.MountPath))
		if err != nil {
			return fmt.Errorf("kubernetes authentication failed: %w", err)
		}

		if err := k.client.SetToken(resp.Auth.ClientToken); err != nil {
			return fmt.Errorf("token set failed: %w", err)
		}

		goto CLIENT_AUTH_SUCCESS
	}

	return errors.New("no authentication settings provided")

CLIENT_AUTH_SUCCESS:

	return nil
}

func (k *Vault) Get(key string) (any, error) {
	match := secretKeyPattern.FindStringSubmatch(key)
	if len(match) != 3 {
		return nil, errors.New("key request does not match the pattern")
	}

	path, secret := match[1], match[2]
	if len(k.PathPrefix) > 0 {
		path = k.PathPrefix + "/" + path
	}

	return k.secretFunc(path, secret)
}

func (k *Vault) readV2(path, secret string) (any, error) {
	response, err := k.client.Secrets.KvV2Read(context.Background(), path, 
		vault.WithMountPath(k.MountPath),
		vault.WithNamespace(k.Namespace),
	)
	if err != nil {
		return nil, err
	}

	if _, ok := response.Data.Data[secret]; !ok {
		return nil, fmt.Errorf("secret not found: %v#%v", path, secret)
	}

	return response.Data.Data[secret], nil
}

func (k *Vault) readV1(path, secret string) (any, error) {
	response, err := k.client.Secrets.KvV1Read(context.Background(), path, 
		vault.WithMountPath(k.MountPath),
		vault.WithNamespace(k.Namespace),
	)
	if err != nil {
		return nil, err
	}

	if _, ok := response.Data[secret]; !ok {
		return nil, fmt.Errorf("secret not found: %v#%v", path, secret)
	}

	return response.Data[secret], nil
}

func (k *Vault) Close() error {
	return nil
}

func init() {
	plugins.AddKeykeeper("vault", func() core.Keykeeper {
		return &Vault{
			KvVersion: "v2",
			Approle: Approle{
				MountPath: "approle",
			},
			K8s: K8s{
				MountPath: "kubernetes",
				TokenPath: "/var/run/secrets/kubernetes.io/serviceaccount/token",
			},
			TLSClientConfig: &pkgtls.TLSClientConfig{},
		}
	})
}
