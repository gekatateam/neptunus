# File Lookup Plugin

The `file` lookup stores configured file content as a lookup data. This plugin requires parser and only first parsed event will be used.

## Configuration
```toml
[[lookups]]
  [lookups.file]
    alias = "file.roles_whitelist"

    # path to file
    file = "roles_whitelist.json"

    # lookup update interval
    # if zero, plugin reads file only on pipeline startup
    interval = "30s"

    [lookups.file.parser]
      type = "json"
```
