# Deduplicate Processor Plugin

The `deduplicate` processor plugin uses Redis to filter duplicates using configured `idempotency_key`. 

Processor adds `::duplicate` label to event with with `true` value if duplicate was found, otherwise, if event has no configured label or if an error occurs during a request to Redis, it will be `false`.

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
```
