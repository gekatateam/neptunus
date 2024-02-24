# Vault Keykeeper Plugin

The `vault` keykeeper plugins allows to read configuration keys from [Hashicorp Vault](https://developer.hashicorp.com/vault). 

Key request format is `%path to secret without mount path%#%secret key%`. 

For example, with `neptunus/kv` engine, secret path `staging/inputs` and `kafka_password` key:
 - request should be `staging/inputs#kafka_password`;
 - plugin `mount_path` parameter - `neptunus/kv`;
 - optionally, you can set `path_prefix` parameter to `staging` - without trailing and leading slashes - and write request as `inputs#kafka_password`.

# Configuration
```toml
[[keykeepers]]
  [keykeepers.vault]
    alias = "vault"

    # Vault address
    address = "http://localhost:8200"

    # Engine mount path
    mount_path = "neptunus/kv"

    # Secrets path prefix
    # May be useful for changing environments using one parameter
    # Instead of change it in each key request
    path_prefix = "test"

    # Kv engine version, "v1" or "v2"
    kv_version = "v2"

    # Vault namespace
    # https://developer.hashicorp.com/vault/tutorials/enterprise/namespace-structure
    namespace = "my-neptunus-namespace"

    ## TLS configuration
    # if true, TLS client will be used
    tls_enable = false
    # trusted root certificates for server
    tls_ca_file = "/etc/neptunus/ca.pem"
    # used for TLS client certificate authentication
    tls_key_file = "/etc/neptunus/key.pem"
    tls_cert_file = "/etc/neptunus/cert.pem"
    # send the specified TLS server name via SNI
    tls_server_name = "exmple.svc.local"
    # use TLS but skip chain & host verification
    tls_insecure_skip_verify = false

    ## Authentication settings
    ## if approle.role_id and approle.secret_id is set, approle method used
    ## if k8s.role and k8s.token_path set, kubernetes method used
    #
    # Approle authentication settings
    # https://developer.hashicorp.com/vault/docs/auth/approle
    [keykeepers.vault.approle]
      mount_path = "approle"
      role_id = "0a72eb67-..."
      secret_id = "e58e2b0d-..."

    # Kubernetes authentication settings
    # https://developer.hashicorp.com/vault/docs/auth/kubernetes
    [keykeepers.vault.k8s]
      mount_path = "kubernetes"
      role_name = "neptunus-role"
      token_path = "/var/run/secrets/kubernetes.io/serviceaccount/token"
```
