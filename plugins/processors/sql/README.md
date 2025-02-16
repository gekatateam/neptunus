The `sql` processor plugin performs SQL query using incoming events. This plugin based on [jmoiron/sqlx](https://github.com/jmoiron/sqlx) package.

An event routing key may be used as a table name using `table_placeholder` parameter.

If query returns rows, it will be added to event.

## TLS usage
Drivers use plugin TLS configuration.

## Configuration
```toml
[[processors]]
  [processors.sql]
    # SQL driver, must be on of: "pgx", "mysql", "sqlserver", "oracle", "clickhouse"
    driver = "pgx"

    # datasource service name in selected driver format
    dsn = "postgres://postgres:pguser@localhost:5432/postgres"

    # if true, one SQL client is shared among the processors in set
    # otherwise, each plugin uses a personal client
    shared = true

    # database connection params - https://pkg.go.dev/database/sql#DB.SetConnMaxIdleTime
    conns_max_idle_time = "10m"
    conns_max_life_time = "10m"
    conns_max_open = 2
    conns_max_idle = 1

    # queries execution timeout
    query_timeout = "10s"

    # a placeholder in query, which will be replaced by event routing key
    # that may be useful if target table is partitioned
    table_placeholder = ":table_name"

    # maximum number of attempts to execute query
    # before event will be marked as failed
    retry_attempts = 0 # zero for endless attempts

    # interval between retries to (re-)execute query
    retry_after = "5s"

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

    # "query placeholders -> event fields" mapping
    [processors.sql.columns]
      expedition_type = "type"
      expedition_region = "region"

    # "event fields -> query result columns" mapping
    # please note that each field type is always a slice
    [processors.sql.fields]
      type = "expedition_type"
      region = "expedition_region"

    # query, that will be executed for event
    [outputs.sql.on_event]
      query = '''
UPDATE :table_name SET
    EXPEDITION_TYPE = :expedition_type
    , EXPEDITION_REGION = :expedition_region
RETURNING EXPEDITION_OWNER, UID;
'''
```
