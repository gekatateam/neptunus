# Redis Processor Plugin

The `redis` processor plugin processes configured `command` and saves it's result in `result_to` event field.

Due to [Redis protocol spec](https://redis.io/docs/latest/develop/reference/protocol-spec/), command writes as an array of **command name** and **it's args**.

You can use incoming event fields in any element of command using `{{- path.to.field -}}` syntax. Please note, that it is **not a template**, just a special pattern.

If field is a slice, each element will be added to command sequence. If field is a map, plugin adds each key-value pairs to command. Other fields will be used as-is.

In configuration example below, this event body:
```json
{
  "keysnum": 3,
  "data": {
    "key": "test data",
    "another key": true,
    "more keys": 1337.42
  }
}
```
will be rendered to: `[ "hsetex", "testspace:map", "ex", 3600, "fields", 3, "key", "test data", "another key", true, "more keys", 1337.42 ]`.

## Configuration
```toml
[[processors]]
  [processors.redis]
    # list of Redis nodes
    servers = [ "localhost:6379" ]

    # Redis credentials
    username = ""
    password = ""

    # operations timeout
    timeout = "30s"

    # maximum number of attempts to execute command
    retry_attempts = 0 # zero for endless attempts

    # interval between retries
    retry_after = "5s"

    # command to execute
    command = [ "hsetex", "testspace:map", "ex", 3600, "fields", "{{- keysnum -}}", "{{- data -}}" ]

    # path to save command result
    # only used if not empty
    result_to = "result"

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
