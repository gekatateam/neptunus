# Http Output Plugin

The `http` output plugin writes events using HTTP client to configured host, but request path depends on event routing key. This plugin requires serializer.

Plugin creates one requester per each unique event routing key, with personal batch controller. By the way, HTTP client shares between requesters.

# Configuration
```toml
[[outputs]]
  [outputs.http]
    # target host, required
    host = "http://localhost:9100"

    # request method, required
    method = "POST"

    # label, which value will be used as a request method, if configured
    # if not, or event has no label, `method` will be used
    # ONLY FIRST EVENT IN BATCH USED
    method_label = "request_method"

    # list of fallback hosts
    # if all `retry_attempts` to perform request to `host` failed, fallbacks will be used
    fallbacks = [ "http://fallback.local:9100", "http://anotherfallback.local:9100" ]

    # time limit for requests made by client
    # zero means no limit
    timeout = "10s"

    # maximum amount of time an idle (keep-alive) connection will remain idle before closing itself
    # zero means no limit
    idle_conn_timeout = "1m"

    # maximum number of idle (keep-alive) connections across all hosts
    # zero means no limit
    max_idle_conns = 10

    # time after which inactive requesters will be closed
    # if configured value a zero, idle requesters will never be closed
    # if configured value less than 1m but not zero, it will be set to 1m
    idle_timeout = "1h"

    # result codes, means request performed successfully
    success_codes = [ 200, 201, 204 ]

    # regular expression to match response body
    # if configured, response code must be in `success_codes` OR body must match this regexp
    success_body = ".*success but with strange code.*"

    # interval between events buffer flushes if buffer length less than it's capacity
    batch_interval = "5s"

    # events buffer size
    # this plugin uses all events serialization result as a request body
    batch_buffer = 100    

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
    [outputs.http.headers]
      authorization = "@{envs:BEARER_TOKEN}"

    # a "header <- label" map
    # if event label exists, it will be added as a request header
    # if "headers" already has same one, it will be overwritten
    # ONLY FIRST EVENT IN BATCH USED
    [outputs.http.headerlabels]
      custom_header = "my_label_name"

    # a "param <- field" map
    # if event field exists, it will be added as a request urlparam
    # target field must not be a map, but slices allowed, if slice not contains maps
    # ONLY FIRST EVENT IN BATCH USED
    [outputs.http.paramfields]
      custom_param = "my.field.path"

    [outputs.http.serializer]
      type = "json"
      data_only = true
```
