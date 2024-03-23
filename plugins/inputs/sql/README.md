# Sql Input Plugin

The `sql` input plugin performs SQL query for reading events. This plugin based on [jmoiron/sqlx](https://github.com/jmoiron/sqlx) package.

## Poll cycle

This plugin works in poll cycle:
1. (if configured) on initialization, plugin executes `on_init` query and caches configured `keep_values`
2. plugin executes `on_poll` query; each row is turned in an event; plugin caches configured `keep_values`
3. (if configured) plugin waits for batch delivery and executes `on_done` query if `on_poll` was successfull

Next cycle will start from second step immediately or each configured `interval`.

## Configuration
```toml
[[inputs]]
  [inputs.sql]
    # SQL driver, must be on of: "pgx", "mysql", "sqlserver", "oracle", "clickhouse"
    driver = "pgx"

    # datasource service name in selected driver format
    dsn = "postgres://postgres:pguser@localhost:5432/postgres"

    # poll interval
    # if zero, next poll cycle will start immediately
    interval = "5s"

    # if true, onDone query will be executed only after all events have been delivered
    wait_for_delivery = true

    # database connection params - https://pkg.go.dev/database/sql#DB.SetConnMaxIdleTime
    conns_max_idle_time = "10m"
    conns_max_life_time = "10m"
    conns_max_open = 2
    conns_max_idle = 1

    # if true, onPoll and onDone queries will be executed in one transaction
    transactional = false

    # transaction isolation level
    # "Default", "ReadUncommitted", "ReadCommitted", "WriteCommitted", 
    # "RepeatableRead", "Snapshot", "Serializable", "Linearizable"
    isolation_level = "Default"

    # is transaction are read-only
    read_only = false

    # if configured, an event id will be set by data from path
    # expected format - "type:path"
    id_from = "field:path.to.id"

    # a "label name -> column name" map
    # if column exists and can be mapped to string type, it will be saved as configured label
    [inputs.sql.labelcolumns]
      event_type = "type"

    # list of columns whose values will be saved for use in queries
    # "first" - only values from first row will be saved
    # "last" - only values from last row will be saved
    # "all" - all values will be saved, one slice per column
    #
    # these settings are applied to init and poll queries
    # it is okay if query does not return configured column
    #
    # keeped values can be used in poll and done queries using named params
    # https://jmoiron.github.io/sqlx/#namedParams
    [inputs.sql.keep_values]
      first = []
      last = [ "insert_timestamp" ]
      all = [ "id" ]

    # initializing query, executed once on plugin startup
    # if both, "file" and "query" are set, file is prioritized
    [inputs.sql.on_init]
      file = "init.sql"
      query = '''
SELECT INSERT_TIMESTAMP FROM POLLING_TABLE
WHERE POLLED_TIMESTAMP IS NULL
ORDER BY INSERT_TIMESTAMP ASC
LIMIT 1;
      '''

    # polling query, executed on each poll cycle
    # this query can use previously keeped values 
    [inputs.sql.on_poll]
      file = "poll.sql"
      query = '''
SELECT ID, INSERT_TIMESTAMP, MESSAGE FROM POLLING_TABLE
WHERE POLLED_TIMESTAMP IS NULL
AND INSERT_TIMESTAMP >= :insert_timestamp
ORDER BY INSERT_TIMESTAMP ASC
LIMIT 100
FOR UPDATE SKIP LOCKED;
      '''

    # final query, executed in the end of each poll cycle
    # this query can use previously keeped values 
    [inputs.sql.on_done]
      file = "done.sql"
      query = '''
UPDATE POLLING_TABLE
SET POLLED_TIMESTAMP = now()
WHERE ID IN (:id)
AND INSERT_TIMESTAMP >= :insert_timestamp;
      '''
```
