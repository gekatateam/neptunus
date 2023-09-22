# Httpl Input Plugin

The `httpl` input plugin serves requests on configured address and streams request body to parser line by line. This plugin requires parser.

If all body parsed without errors, plugin returns `200 OK` with `accepted events: N` body. If reading error occures, plugin returns `500 Internal Server Error`, if parsing error occures, it's `400 Bad Request`.

This plugin produce events with routing key as request path, `server` label with configured address and `sender` label with request RemoteAddr address.

> **Note**
> This plugin may write it's own [metrics](../../../docs/METRICS.md#summary-plugin_http_server_requests_seconds)

## Configuration
```toml
[[inputs]]
  [inputs.httpl]
    # if true, plugin server writes it's own metrics
    enable_metrics = false

    # address and port to host HTTP listener on
    address = ":9200"

    # number of maximum simultaneous connections
    max_connections = 10

    # maximum duration before timing out read of the request
    read_timeout = "10s"

    # maximum duration before timing out write of the response
    write_timeout = "10s"

    # a "label name -> header" map
    # if request header exists, it will be saved as configured label
    [inputs.httpl.labelheaders]
      length = "Content-Length"
      encoding = "Content-Type"

    [inputs.httpl.parser]
      type = "json"
```
