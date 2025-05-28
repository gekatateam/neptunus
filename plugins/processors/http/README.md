# Http Processor Plugin

The `http` processor plugin performs HTTP requests for each event. This plugin requires parser and serializer.

Unlike [HTTP output](../../outputs/http/README.md), this plugin uses configured label as a request path (it's optional, btw), not event routing key.

Please note, that in multiline configuration HTTP client is shared between processors in set.

# Configuration
```toml
[[processors]]
  [processors.http]
    # target host, required
    host = "http://localhost:9100"

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

    # label, which value will be used as a request path, if configured
    path_label = "request_uri"

    # field, which content will be used as a request body after serialization, if configured
    request_body_from = "path.to.request.field"

    # field, that will be added to event with response body after parsing, if configured
    response_body_to = "path.to.response.field"

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

    # a "header -> label" map
    # if event label exists, it will be added as a request header
    [processors.http.headerlabels]
      custom_header = "my_label_name"

    # a "param -> field" map
    # if event field exists, it will be added as a request urlparam
    # target field must not be a map, but slices allowed, if slice not contains maps
    [processors.http.paramfields]
      custom_param = "my.field.path"

    [processors.http.serializer]
      type = "json"
      data_only = true

    [processors.http.parser]
      type = "json"
      split_array = true
```
