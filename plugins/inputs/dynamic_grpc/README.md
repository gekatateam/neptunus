# Dynamic gRPC Input Plugin

The `dynamic_grpc` input can read server stream as a client or receive unary calls/client streams as a server. Each received message will be decoded using [protomap](https://github.com/gekatateam/protomap) to exactly one event, which routing key is a full name of procedure.

## Configuration
```toml
[[inputs]]
  [inputs.dynamic_grpc]
    # plugin mode, "ServerSideStream" or "AsServer"
    mode = "ServerSideStream"

    # list of .proto files with messages and procedures
    proto_files = [ 'D:\Go\_bin\protos\marketdata.proto' ]

    # list of import paths to resolve .proto imports
    import_paths = [ 'D:\Go\_bin\protos\' ]

    # if configured, an event id will be set by data from path
    # expected format - "type:path"
    id_from = "field:path.to.id"

    # procedures to call/listen
    [[inputs.gynamic_grpc.procedures]]
      # name MUST be server stream in "ServerSideStream" mode
      # and MUST be unary/client stream in "AsServer" mode
      name = 'public.invest.api.contract.v1.MarketDataStreamService.MarketDataServerSideStream'

      # used in "ServerSideStream" mode
      # request body in json that will be encoded to procedure input message
      invoke_request = '''
{
  "subscribe_order_book_request": {
    "subscription_action": 1,
    "instruments": [
      {
        "instrument_id": "F0",
        "depth": 10,
        "order_book_type": 1
      }
    ]
  }
}'''
      # used in "ServerSideStream" mode
      # invoke headers
      invoke_headers = { authorization = "Bearer XXXXXX" }

      # used in "AsServer" mode
      # response body in json that will be used as unary call response/client stream end message
      invoke_response = '''
{
  "accepted": true
}'''

    # a "label name <- header" map
    # if received message header exists, it will be saved as configured label
    [inputs.dynamic_grpc.labelheaders]
      x-ratelimit-limit = "x-ratelimit-limit"

    # gRPC client settings
    # used in "ServerSideStream" mode
    [inputs.dynamic_grpc.client]
      # server address, see more info about uri schemes
      # https://grpc.github.io/grpc/core/md_doc_naming.html
      address = "sandbox-invest-public-api.domain.net:443"

      # interval between retries to (re-)establish a connection
      retry_after = "5s"

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

    # gRPC server settings
    # used in "AsServer" mode
    [inputs.dynamic_grpc.server]
      # address and port to host HTTP/2 listener on
      address = ":9900"

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
