# Grpc Output Plugin

The `grpc` output plugin sends an events to external systems using gRPC. See [input.proto](../../common/grpc/input.proto) for more information. This plugin requires serializer.

Plugin can be configured for using one of three RPCs:
 - `unary` - plugin passes each event in serializer and send it by unary call.
 - `bulk` - plugin sends a stream of data after every `interval` or when events `buffer` is full.
 - `stream` - plugin sends an endless stream of events; when server sends **cancellation token** plugin closes stream, waits for a `sleep` and reconnects. This mode designed for streaming between Neptunes.

> **Note**
> This plugin may write it's own [metrics](../../../docs/METRICS.md#grpc-client)

## Configuration
```toml
[[outputs]]
  [outputs.grpc]
    # if true, plugin client writes it's own metrics
    enable_metrics = false

    # server address, see more info about uri schemes
    # https://grpc.github.io/grpc/core/md_doc_naming.html
    address = "localhost:5800"

    # procedure to be used by plugin, "unary", "bulk", or "stream"
    procedure = "bulk"

    # interval between retries to (re-)establish a connection
    retry_after = "5s"

    # maximum number of attempts of unary calls/to reopen streams
    # before the event will be marked as failed
    max_attempts = 0 # zero for endless attempts

    ## batching settings, using only in "bulk" mode
    # interval between sending event batches if buffer length less than it's capacity
    batch_interval = "5s"
    # events buffer size
    batch_buffer = 100

    ## TLS configuration
    # if true, TLS client will be used
    tls_enable = false
    # trusted root certificates for server
    tls_ca_file = "/etc/neptunus/ca.pem"
    # used for TLS client certificate authentication
    tls_key_file = "/etc/neptunus/key.pem"
    tls_cert_file = "/etc/neptunus/cert.pem"
    # send the specified TLS server name via SNI
    tls_server_name = "exmple.svc.local"
    # use TLS but skip chain & host verification
    tls_insecure_skip_verify = false

    # connections set up parameters
    [outputs.grpc.dial_options]
      # if set, value will be used as the :authority pseudo-header 
      # and as the server name in authentication handshake
      authority = ""

      # specifies a user agent string for all the RPCs
      user_agent = ""

      # client keepalive options
      # see more in https://pkg.go.dev/google.golang.org/grpc/keepalive#ClientParameters
      inactive_transport_ping = "0s" # zero is for infinity
      inactive_transport_age = "20s"
      permit_without_stream = false

    # calls configuration, applies to each RPC
    [outputs.grpc.call_options]
      # set the content-subtype for a call
      content_subtype = ""

      # configures the action to take when an RPC is attempted on 
      # broken connections or unreachable servers
      wait_for_ready = false

    # a "metadata -> label" map
    # if event label exists, it will be added as a call metadata
    # used in "one" mode only
    [outputs.grpc.metadatalabels]
      custom_header = "my_label_name"

    [outputs.grpc.serializer]
      type = "json"
      data_only = true
```
