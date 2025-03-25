# Httpl Input Plugin

The `httpl` input plugin serves requests on configured address and streams request body to parser line by line, so, unlike [http input](../http/), this plugin is good for streaming. This plugin requires parser.

If all body parsed without errors, plugin returns `200 OK` with `accepted events: N` body. If reading error occures, plugin returns `500 Internal Server Error`, if parsing error occures, it's `400 Bad Request`.

This plugin produce events with routing key as request path, `server` label with configured address and `sender` label with request RemoteAddr address.

> [!TIP]  
> This plugin may write it's own [metrics](../../../docs/METRICS.md#http-server)

## Configuration
```toml
[[inputs]]
  [inputs.httpl]
    # if true, plugin server writes it's own metrics
    enable_metrics = false

    # address and port to host HTTP listener on
    address = ":9800"

    # number of maximum simultaneous connections
    max_connections = 10

    # maximum duration before timing out read of the request
    read_timeout = "10s"

    # maximum duration before timing out write of the response
    write_timeout = "10s"

    # optional basic auth credentials
    # if both set, requests with wrong credentials will be rejected with `401 Unauthorized`
    basic_isername = "user"
    basic_password = "pass"

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

    # a "label name -> header" map
    # if request header exists, it will be saved as configured label
    [inputs.httpl.labelheaders]
      length = "Content-Length"
      encoding = "Content-Type"

    [inputs.httpl.parser]
      type = "json"
```
