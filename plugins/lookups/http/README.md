# HTTP Lookup Plugin

The `http` lookup stores HTTP response body as a lookup data. This plugin requires parser and only first parsed event will be used.

## Configuration
```toml
[[lookups]]
  [lookups.http]
    alias = "cloud_token"

    # lookup update interval
    interval = "30s"

    # target host, required
    host = "http://localhost:9100"

    # list of fallback hosts
    # if all `retry_attempts` to perform request to `host` failed, fallbacks will be used
    fallbacks = [ "http://fallback.local:9100", "http://anotherfallback.local:9100" ]

    # request method, required
    method = "POST"

    # time limit for requests made by client
    # zero means no limit
    timeout = "10s"

    # maximum amount of time an idle (keep-alive) connection will remain idle before closing itself
    # zero means no limit
    idle_conn_timeout = "1m"

    # maximum number of idle (keep-alive) connections across all hosts
    # zero means no limit
    max_idle_conns = 10

    # result codes, means request performed successfully
    success_codes = [ 200, 201, 204 ]

    # regular expression to match response body
    # if configured, response code must be in `success_codes` OR body must match this regexp
    success_body = ".*success but with strange code.*"

    # request body
    request_body = ""

    # request query
    request_query = "?client_secret=XXX&client_key=YYY"

    # maximum number of attempts to perform request
    # before events will be marked as failed
    # 
    # only requests that ended with `success_codes`
    # will NOT be retried
    retry_attempts = 0 # zero for endless attempts

    # interval between retries to perform bulk request
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

    # static headers that will be used on each request
    [lookups.http.headers]
      authorization = "@{envs:BEARER_TOKEN}"

    [lookups.http.parser]
      type = "json"
```
