# Http Input Plugin

The `http` input plugin serves requests on configured address. This plugin requires parser.

Unlike [httpl input](../httpl/), this plugin fully reads request body before parsing stage, and also takes into account URL-encoded request params.

Plugin behaviour can be configured using `on_*` params.

This plugin produce events with routing key as request path (or matched pattern if `paths` configured), `server` label with configured address, `sender` label with request RemoteAddr address and `method` with request HTTP method.

> [!TIP]  
> This plugin may write it's own [metrics](../../../docs/METRICS.md#http-server)

## Configuration
```toml
[[inputs]]
  [inputs.http]
    # if true, plugin server writes it's own metrics
    enable_metrics = false

    # address and port to host HTTP listener on
    address = ":9900"

    # optional paths to listen
    # if not configured, `/` will be used
    paths = [ "/http.input/{uid}/push", "/http.input/{id}/put" ]

    # optional path values to set as events labels if exists
    path_values = [ "uid", "id" ]

    # number of maximum simultaneous connections
    max_connections = 10

    # maximum duration before timing out read of the request
    read_timeout = "10s"

    # maximum duration before timing out write of the response
    write_timeout = "10s"

    # optional basic auth credentials
    # if both set, requests with wrong credentials will be rejected with `401 Unauthorized`
    basic_username = "user"
    basic_password = "pass"

    # allowed incoming requests methods, POST, PUT, GET
    allowed_methods = [ "POST", "PUT" ]

    # if not empty, plugin will try to save request parameters 
    # in events body at the specified path
    query_params_to = "."

    # if true, server waits for events to be delivered
    # before responding to a client
    wait_for_delivery = false

    # if configured, an event id will be set by data from path
    # expected format - "type:path"
    id_from = "field:path.to.id"

    ## TLS configuration
    # if true, TLS listener will be used
    tls_enable = false
    # service key and certificate
    tls_key_file = "/etc/neptunus/key.pem"
    tls_cert_file = "/etc/neptunus/cert.pem"
    # one or more allowed client CA certificate file names to
    # enable mutually authenticated TLS connections
    tls_allowed_cacerts = [ "/etc/neptunus/clientca.pem" ]
    # minimum and maximum TLS version accepted by the service
    # not limited by default
    tls_min_version = "TLS12"
    tls_max_version = "TLS13"

    # response if all events successfully accepted by plugin
    # body may be an fmt template where param is an events count
    [inputs.http.on_success]
      code = 200
      body = "accepted events: %v"

    # response if body parsing failed
    # body may be an fmt template where param is error returned by parser
    [inputs.http.on_parser_error]
      code = 400
      body = "parsing failed: %v"

    # response if request method is not in "allowed_methods"
    [inputs.http.on_parser_error]
      code = 405
      body = "method not allowed"

    # response if basic auth failed
    [inputs.http.on_parser_error]
      code = 401
      body = "unauthorized"

    # response if any other error occurred
    # body may be an fmt template where param is occurred error
    [inputs.http.on_other_error]
      code = 500
      body = "internal error occured: %v"

    # a "label name <- header" map
    # if request header exists, it will be saved as configured label
    [inputs.http.labelheaders]
      length = "Content-Length"
      encoding = "Content-Type"

    [inputs.http.parser]
      type = "json"
```
