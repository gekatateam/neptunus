# Protobuf Parser Plugin

The `protobuf` parser plugin can be used to decode Protocol Buffers-encoded binary data into a `map[string]any` event body. This parser always produces one event.

## Configuration
```toml
[[inputs]]
  [inputs.http.parser]
    type = "protobuf"

    # list of .proto files with target message schema and it's dependencies
    proto_files = [ ".pipelines/payload.proto" ]

    # list of import paths to resolve .proto imports
    import_paths = [ 'D:\Go\_bin\protos\' ]

    # full message name, which schema will be used to decode input binary
    message = "protomap.test.Test"
```
