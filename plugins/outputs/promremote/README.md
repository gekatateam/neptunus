# Promremote Output Plugin

The `promremote` plugin writes events as metrics using [Prometheus remote-write protocol](https://prometheus.io/docs/specs/prw/remote_write_spec/).

This plugin uses event model from [stats processor](../../processors/stats/):
 - each event must have `::name` label - which is used in `__name__` metric label
 - each event must have `stats` field, and it's type must be `map[string]number` where `number` - value, that can be converted to `float64`

For each event, the plugin creates as many metrics as the number of keys contained in the `stats` field that were successfully converted to `float64`.

Each metric name creates as `::name` label plus `_%stats subkey%`.

Metric labels takes from event labels, excluding configured `ignore_labels`. 

Metrics timestamp takes from events timestamp.

For example, with event:
```json
{
  "id": "af002295-7c47-4323-ae5f-f268fad56340",
  "routing_key": "neptunus.generated.metric",
  "timestamp": "2023-08-25T22:29:28.9120822+00:00",
  "tags": [],
  "labels": {
    "::line": "3",
    "region": "US/California",
    "::type": "metric",
    "::name": "traffic.now"
  },
  "data": {
    "stats": { 
      "count": 11,
      "sum": 125,
      "avg": 11.9
    }
  }
}
```

metrics will be:
```
traffic.now_count{::line="3", region="US/California"} 11.0 1692991768912
traffic.now_sum{::line="3", region="US/California"} 125.0 1692991768912
traffic.now_avg{::line="3", region="US/California"} 11.9 1692991768912
```

# Configuration
```toml
[[outputs]]
  [outputs.promremote]
    # remote write endpoint, required
    host = "http://localhost:8428/api/v1/write"

    # time limit for requests made by client
    # zero means no limit
    timeout = "10s"

    # maximum amount of time an idle (keep-alive) connection will remain idle before closing itself
    # zero means no limit
    idle_conn_timeout = "1m"

    # list of event labels, that will be ignored
    ignore_labels = [ "::type", "::name" ]

    # interval between events buffer flushes if buffer length less than it's capacity
    batch_interval = "5s"

    # events buffer size
    batch_buffer = 100    

    # maximum number of attempts to execute request
    # before events will be marked as failed
    # 
    # only requests that ended with `2xx`
    # will NOT be retried
    retry_attempts = 0 # zero for endless attempts

    # interval between retries to execute bulk request
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

    # a "header -> label" map
    # if event label exists, it will be added as a request header
    # ONLY FIRST EVENT IN BATCH USED
    [outputs.promremote.headerlabels]
      custom_header = "my_label_name"
```
