# Deduplicate Processor Plugin

The `deduplicate` processor plugin uses Redis to filter duplicates using configured `idempotency_key`. 

Processor adds `::duplicate` event with with `true` value if duplicate was found, otherwise, if event has no configured label or if an error occurs during a request to Redis, it will be `false`.

## Configuration
```toml
[[processors]]
  [processors.deduplicate]
    # event label whose value will be used for deduplication
    idempotency_key = "unique_key"

    # maximum number of attempts to execute duplicate search op
    retry_attempts = 0 # zero for endless attempts

    # interval between retries
    retry_after = "5s"

    # Redis connection config
    [processors.deduplicate.redis]
      # if true, one Redis client is shared among the processors in set
      # otherwise, each plugin uses a personal client
      shared = true

      # list of Redis nodes
      servers = [ "localhost:6379" ]

      # Redis credentials
      username = ""
      password = ""

      # operations timeout
      timeout = "30s"

      # keys TTL
      ttl = "1h"      

      # Redis keyspace
      # keys are stored in %keyspace%:%idempotency_key value% format
      keyspace = "neptunus:deduplicate"

      # connection pool settings
      conns_max_open = 2
      conns_max_idle = 1
      conns_max_life_time = "10m"
      conns_max_idle_time = "10m"
```
