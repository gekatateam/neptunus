# Protobuf Serializer Plugin

The `protobuf` serializer plugin can be used to encode event data to protobuf binary.

> [!CAUTION]
> This plugin **always** accepts exactly one event, and event data **must** be `map[string]any`

## Configuration
```toml
[[outputs]]
  [outputs.http.serializer]
    type = "protobuf"

    # list of .proto files with target message schema and it's dependencies
    proto_files = [ ".pipelines/payload.proto" ]

    # list of import paths to resolve .proto imports
    import_paths = [ 'D:\Go\_bin\protos\' ]

    # full message name, which schema will be used to encode data
    message = "protomap.test.Test"
```
