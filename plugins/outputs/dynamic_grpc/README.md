# Dynamic gRPC Output Plugin

The `dynamic_grpc` output can produce events using unary RPCs or client streams in client mode, and produce events to subscribers through server streams in server mode.

## Client mode

Event routing key must be a full RPC name from `proto_files`. If RPC:
 - unary - an individual call is made for each event;
 - client stream - **one** stream is performed for each batch of events.

Plugin creates one caller per each unique event routing key, with personal batch controller. By the way, gRPC client shares between callers.

Each event will be encoded using [protomap](https://github.com/gekatateam/protomap). Concrete message descriptor takes from RPC input.

## Server mode

Event routing key must be a full RPC name from `procedures` and must be declared in  `proto_files`.

Plugin creates one publisher per each unique event routing key.

> [!WARNING]  
> If there is no active streams, event will be dropped. This mode was designed for realtime notifications.

Each event will be encoded using [protomap](https://github.com/gekatateam/protomap). Concrete message descriptor takes from RPC output.

## Configuration
```toml
[[outputs]]
  [outputs.dynamic_grpc]
    # plugin mode
    # "AsClient" or "AsServer"
    mode = "AsClient"

    # list of .proto files with messages and procedures to call/listen
    proto_files = [ 'D:\Go\_bin\protos\marketdata.proto' ]

    # list of import paths to resolve .proto imports
    import_paths = [ 'D:\Go\_bin\protos\' ]

    # static headers that will be used on each RPC
    # used only in client mode
    [outputs.dynamic_grpc.headers]
      authorization = "@{envs:BEARER_TOKEN}"

    # a "header <- label name" map
    # if event label exists, it will be added to RPC as a header
    # if "headers" already has same one, it will be overwritten
    # used only in client mode
    [outputs.dynamic_grpc.headerlabels]
      x-ratelimit-limit = "x-ratelimit-limit"

    [outputs.dynamic_grpc.client]
      # server address, see more info about uri schemes
      # https://grpc.github.io/grpc/core/md_doc_naming.html
      address = "sandbox-invest-public-api.domain.net:443"

      # time limit for RPCs made by client
      # zero means no limit
      invoke_timeout = "30s"

      # time after which inactive callers will be closed
      # if configured value a zero, idle callers will never be closed
      # if configured value less than 1m but not zero, it will be set to 1m
      idle_timeout = "1h"

      # status codes, means RPC performed successfully
      # https://grpc.io/docs/guides/status-codes/#the-full-list-of-status-codes
      success_codes = [ 0 ]

      # regular expression to match status message
      # if configured, status code must be in `success_codes` OR status message must match this regexp
      success_message = ".*success but with strange code.*"

      # interval between retries to (re-)establish a connection/call RPC
      retry_after = "5s"

      # maximum number of attempts to make unary calls/(re-)open streams
      # before the event (or whole batch, if it is a stream) will be marked as failed
      retry_attempts = 0 # zero for endless attempts

      # interval between sending batches if buffer length less than it's capacity
      batch_interval = "5s"

      # events buffer size
      batch_buffer = 100

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

    [outputs.dynamic_grpc.server]
      # address and port to host HTTP/2 listener on
      address = ":9900"

      # publishers behaviour
      # "Random" - message will be sent to a random open stream
      # "Broadcast" - message will be sent to every open stream
      behaviour = "Random"

      # list of procedures to listen
      # each must be a server-side stream
      procedures = [ "neptunus.plugins.common.grpc.Input.SendServerStream" ]

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
```
