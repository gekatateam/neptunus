# Dynamic gRPC Input Plugin

The `dynamic_grpc` input can read server-side stream as a client. Each received message will be decoded using [protomap](https://github.com/gekatateam/protomap) to exactly one event, which routing key is a full name of procedure.

Ability to receive unary calls and client streams as a server will be implemented in future releases.

## Configuration
```toml
[[inputs]]
  [inputs.dynamic_grpc]
    # plugin mode, right now it is only "ServerSideStream"
    # procedure to call MUST be server-side stream
    mode = "ServerSideStream"

    # list of .proto files with messages and procedure to call
    proto_files = [ 'D:\Go\_bin\protos\marketdata.proto' ]

    # list of import paths to resolve .proto imports
    import_paths = [ 'D:\Go\_bin\protos\' ]

    # procedure name to call
    procedure = 'public.invest.api.contract.v1.MarketDataStreamService.MarketDataServerSideStream'

    # gRPC client settings
    # used in "ServerSideStream" mode
    [inputs.dynamic_grpc.client]
      # server address, see more info about uri schemes
      # https://grpc.github.io/grpc/core/md_doc_naming.html
      address = "sandbox-invest-public-api.domain.net:443"

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

      # json-encoded request that will be encoded to procedure input message
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

      # invoke headers
      [inputs.dynamic_grpc.client.invoke_headers]
        authorization = "Bearer "

    # a "label name <- header" map
    # if received message header exists, it will be saved as configured label
    [inputs.dynamic_grpc.labelheaders]
      x-ratelimit-limit = "x-ratelimit-limit"
```
