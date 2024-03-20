# Sql Input Plugin

The `sql` input plugin performs SQL query for reading events. This plugin based on [jmoiron/sqlx](https://github.com/jmoiron/sqlx) package.

## Poll cycle

This plugin works in poll cycle:
0. (if configured) on initialization, plugin executes `on_init` query and caches configured `keep_values`
1. plugin executes `on_poll` query; each row is turned in an event; plugin caches configured `keep_values`
2. (if configured) plugin waits for batch delivery and executes `on_done` query if `on_poll` was successfull

If configured, steps 1 and 2 executing in transaction.

## Configuration
```toml
[[inputs]]
  [inputs.sql]
    driver = "pgx"
    dsn = "postgres://postgres:pguser@localhost:5432/postgres"
    interval = "5s"
    wait_for_delivery = true
    transactional = true
    [inputs.sql.keep_values]
      last = [ "insert_timestamp" ]
      all = [ "id" ]
    [inputs.sql.on_init]
      query = '''
SELECT INSERT_TIMESTAMP FROM POLLING_TABLE
WHERE POLLED_TIMESTAMP IS NULL
ORDER BY INSERT_TIMESTAMP ASC
LIMIT 1;
      '''
    [inputs.sql.on_poll]
      query = '''
SELECT ID, INSERT_TIMESTAMP, MESSAGE FROM POLLING_TABLE
WHERE POLLED_TIMESTAMP IS NULL
AND INSERT_TIMESTAMP >= :insert_timestamp
ORDER BY INSERT_TIMESTAMP ASC
LIMIT 50
FOR UPDATE SKIP LOCKED;
      '''
    [inputs.sql.on_done]
      query = '''
UPDATE POLLING_TABLE
SET POLLED_TIMESTAMP = now()
WHERE ID IN (:id)
AND INSERT_TIMESTAMP >= :insert_timestamp;
      '''
```
