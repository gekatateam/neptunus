# Sql Lookup Plugin

The `sql` lookup plugin performs SQL query for reading lookup data. This plugin is based on [jmoiron/sqlx](https://github.com/jmoiron/sqlx) package.



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
