# Sql Output Pluign

The `sql` output plugin performs SQL query for writing events. This plugin based on [jmoiron/sqlx](https://github.com/jmoiron/sqlx) package.

Plugin creates one producer per each unique event routing key. An event routing key may be used as a table name using `table_placeholder` parameter.

## TLS usage
Drivers use plugin TLS configuration.

## Configuration
```toml
[[outputs]]
  [outputs.sql]
    # SQL driver, must be on of: "pgx", "mysql", "sqlserver", "oracle", "clickhouse"
    driver = "pgx"

    # datasource service name in selected driver format
    dsn = "postgres://postgres:pguser@localhost:5432/postgres"

    # database connection params - https://pkg.go.dev/database/sql#DB.SetConnMaxIdleTime
    # one connection pool shares between producers
    conns_max_idle_time = "10m"
    conns_max_life_time = "10m"
    conns_max_open = 2
    conns_max_idle = 1

    # queries execution timeout
    query_timeout = "10s"

    # a placeholder in push query, which will be replaced by event routing key
    # that may be useful if target table is partitioned
    table_placeholder = ":table_name"

    # time after which inactive producers will be closed
    # if configured value a zero, idle producers will never be closed
    # if configured value less than 1m but not zero, it will be set to 1m
    idle_timeout = "5m"

    # interval between events buffer flushes if buffer length less than it's capacity
    batch_interval = "5s"

    # events buffer size
    # if configured value less than 1, it will be set to 1
    batch_buffer = 100

    # maximum number of attempts to execute push query
    # before events batch will be marked as failed
    retry_attempts = 0 # zero for endless attempts

    # interval between retries to (re-)execute push query
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

    # "push query placeholders -> event fields" mapping
    [outputs.sql.columns]
      expedition_type = "type"
      expedition_region = "region"

    # optional init query, runs once on plugin startup
    # if init query fails, plugin initialization fails too
    [outputs.sql.on_init]
      query = '''
CREATE TABLE IF NOT EXISTS EXPEDITIONS (
	EXPEDITION_TYPE TEXT
	, EXPEDITION_REGION TEXT
);
'''

    # push query, uses all events buffer, so, bulk inserts also available
    [outputs.sql.on_push]
      query = '''
INSERT INTO :table_name (EXPEDITION_TYPE, EXPEDITION_REGION)
VALUES (:expedition_type, :expedition_region)
ON CONFLICT DO NOTHING;
'''
```
