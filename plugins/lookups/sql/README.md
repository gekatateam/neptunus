# Sql Lookup Plugin

The `sql` lookup plugin performs SQL query for reading lookup data. This plugin is based on [jmoiron/sqlx](https://github.com/jmoiron/sqlx) package.

A few words about plugin modes.

In `horizontal` mode plugin stores query result as a list of maps. For example, if your query returns table like this:
```
| role        | description |
| ----------- | ----------- |
| admin       | Full access |
| user        | Just user   |
```

The lookup data will be:
```json
[
  {"role": "admin", "description": "Full access"},
  {"role": "user",  "description": "Just user"},
]
```

More classic case, `vertical`, takes `key_column` column as a map key:
```
| param_name  | value       | description   |
| ----------- | ----------- | ------------- |
| currency    | USDT        | Main currency |
| fee_percent | 2           | Transfer fee  |
```

And transforms query result into map of maps:
```json
{
  "currency":    {"value": "USDT", "description": "Main currency"},
  "fee_percent": {"value": "2",    "description": "Transfer fee"}
}
```

## Configuration
```toml
[[lookups]]
  [lookups.sql]
    alias = "transfers.settings"

    # lookup update interval
    # if zero, plugin executes onUpdate query only on pipeline startup
    interval = "30s"

    # SQL driver, must be on of: "pgx", "mysql", "sqlserver", "oracle", "clickhouse"
    driver = "pgx"

    # datasource service name in selected driver format
    dsn = "postgres://localhost:5432/postgres"

    # authentication credentials
    # if both not empty, takes precedence over ones provided in DSN
    username = ""
    password = ""

    # queries execution timeout
    timeout = "30s"

    # plugin mode, "vertical" or "horizontal"
    mode = "vertical"

    # column name which value will be used as a lookup map key
    # used and required only in "vertical" mode
    key_column = "param_name"

    # database connection params - https://pkg.go.dev/database/sql#DB.SetConnMaxIdleTime
    conns_max_idle_time = "10m"
    conns_max_life_time = "10m"
    conns_max_open = 2
    conns_max_idle = 1

    ## TLS configuration
    # if true, TLS client will be used
    tls_enable = false
    # trusted root certificates for server
    tls_ca_file = "/etc/neptunus/ca.pem"
    # used for TLS client certificate authentication
    tls_key_file = "/etc/neptunus/key.pem"
    tls_cert_file = "/etc/neptunus/cert.pem"
    # minimum TLS version, not limited by default
    tls_min_version = "TLS12"
    # send the specified TLS server name via SNI
    tls_server_name = "exmple.svc.local"
    # use TLS but skip chain & host verification
    tls_insecure_skip_verify = false

    # lookup data query
    # # if both, "file" and "query" are set, file is prioritized
    [lookups.sql.on_update]
      file = "settings.sql"
      query = '''
SELECT * FROM SETTINGS_TABLE;
      '''
```
