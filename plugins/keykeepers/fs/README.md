# FS Keykeeper Plugin

The `fs` keykeeper retrieves configuration keys from files.

## Configuration
```toml
[[keykeepers]]
  [keykeepers.fs]
    alias = "fs"

[[keykeepers]]
  [keykeepers.vault]
    alias = "vault"
    address = "https://vault.local:443"
    [keykeepers.vault.approle]
      role_id = "@{fs:/data/vault_role_id}"
      secret_id = "@{fs:./vault_secret_id}"
```
