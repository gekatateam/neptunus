# Protobuf Parser Plugin

The `protobuf` parser plugin can be used to decode protobuf-encoded binary data to `map[string]any` event body. This parser always produce one event.

## Configuration
```toml
[[inputs]]
  [inputs.http.parser]
    type = "protobuf"

    # list of .proto files with target message schema and it's dependencies
    proto_files = [ ".pipelines/payload.proto" ]

    # full message name, which schema will be used to decode input binary
    message = "protomap.test.Test"
```
