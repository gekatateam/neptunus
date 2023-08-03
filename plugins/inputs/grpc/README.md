# Grpc Input Plugin

The `grpc` input plugin serves RPCs on configured address. See [input.proto](../../common/grpc/input.proto) for more information. This plugin requires parser.

Plugin behavoiur depends on the procedure being called:
 - `SendOne` accepts raw data and passes it to configured parser; if parser returns an error, RPC returns `InvalidArgument` with an error.
 - `SendBulk` accepts stream and passes each message to configured parser; it's strongly NOT recommended to use this procedure for long-living streams, as the server will not shutdown before the RPC finishes. At the end, procedure returns summary with count of accepted and failed events and, if there were errors, a map with message index as a key and occured error as a value. Message index is counted from zero.
 - `SendStream` uses **bidi stream** with **event** as an input and **cancellation token** as an output; received event data is passed to configured parser and uses as event body. This procedure MAY be used for long-lived streams, but it's designed for streaming between Neptunes. When the server is shutting down, it sends a cancellation token, and when a client receives it, it MUST close stream and wait for some time before calling the procedure again.

## Configuration
```toml
[[inputs]]
  [inputs.grpc]
    # address and port to host HTTP/2 listener on
    address = ":5800"

    # gRPC server options
    [inputs.grpc.server_options]
      # max size of input and output messages in bytes
      max_message_size = 4_194_304

      # number of worker goroutines that should be used to process incoming streams
      num_stream_workers = 5

      # limit on the number of concurrent streams to each ServerTransport
      max_concurrent_streams = 5

      # keepalive options
      # see more in https://pkg.go.dev/google.golang.org/grpc/keepalive#ServerParameters
      max_connection_idle = "0s" # zero if for infinity
      max_connection_age = "0s"
      max_connection_grace = "0s"
      inactive_transport_ping = "2h"
      inactive_transport_age = "20s"

  # a "label name -> header" map
  # if request metadata exists, it will be saved as configured label
  # used in SendOne and SendBulk procedures only
  [inputs.grpc.labelmetadata]
    agent = "user-agent"

  [inputs.grpc.parser]
    type = "json"
```
