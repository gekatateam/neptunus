# Grpc Input Plugin

The `grpc` input plugin serves RPCs on configured address. See [input.proto](../../common/grpc/input.proto) for more information. This plugin requires parser.

Plugin behavoiur depends on the procedure being called:
 - `SendOne` accepts raw data and passes it to configured parser; if parser returns an error, RPC returns `InvalidArgument` with an error.
 - `SendBulk` accepts stream and passes each message to configured parser; it's strongly NOT recommended to use this procedure for long-living streams, as the server will not shutdown before the RPC finishes. At the end, procedure returns summary with count of accepted and failed events and, if there were errors, a map with message index as a key and occured error as a value. Message index is counted from zero.
 - `SendStream` uses **bidi stream** with **event** as an input and **cancellation token** as an output; received event data is passed to configured parser and uses as event body. This procedure MAY be used for long-lived streams, but it's **designed for streaming between Neptunes**. When the server is shutting down, it sends a cancellation token, and when a client receives it, it MUST close stream and wait for some time before calling the procedure again.

`SendOne` and `SendBulk` produces events with full procedure name as routing key, `server` label with configured address and `sender` label with peer address, `SendStream` produces events as-is.

> [!TIP]  
> This plugin may write it's own [metrics](../../../docs/METRICS.md#grpc-server)

## Configuration
```toml
[[inputs]]
  [inputs.grpc]
    # if true, plugin server writes it's own metrics
    enable_metrics = false

    # address and port to host HTTP/2 listener on
    address = ":5800"

    # only using in SendOne and SendBulk RPCs
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

    # gRPC server options
    [inputs.grpc.server_options]
      # max size of input and output messages in bytes
      max_message_size = "4MiB"

      # number of worker goroutines that should be used to process incoming streams
      num_stream_workers = 5

      # limit on the number of concurrent streams to each ServerTransport
      max_concurrent_streams = 5

      # server keepalive options
      # see more in https://pkg.go.dev/google.golang.org/grpc/keepalive#ServerParameters
      max_connection_idle = "0s" # zero is for infinity
      max_connection_age = "0s"
      max_connection_grace = "0s"
      inactive_transport_ping = "2h"
      inactive_transport_age = "20s"

    # a "label name <- metadata" map
    # if request metadata exists, it will be saved as configured label
    # used in SendOne and SendBulk procedures only
    [inputs.grpc.labelmetadata]
        agent = "user-agent"

    [inputs.grpc.parser]
        type = "json"
```
