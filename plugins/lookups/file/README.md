# File Lookup Plugin

The `file` lookup stores the content of a configured file as lookup data. This plugin requires a parser, and only the first parsed event is used.

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
