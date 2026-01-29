# Exec Lookup Plugin

The `exec` lookup executes configured command with args and stores the result as lookup data. This plugin requires parser and only first parsed event will be used.

## Configuration
```toml
[[lookups]]
  [lookups.exec]
    alias = "token.renew"

    # command to execute
    command = "roles_whitelist.json"

    # command args, strings list
    args = []

    # exec timeout
    timeout = "10s"

    # lookup update interval
    # if zero, plugin executes command only on pipeline startup
    interval = "30s"

    [lookups.file.parser]
      type = "json"
```
