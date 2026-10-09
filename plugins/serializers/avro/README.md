# Avro Serializer Plugin

The `avro` serializer plugin encodes event data to Avro binary. The Avro schema must be provided in the serializer configuration, and event data must match that schema.

> [!CAUTION]
> This plugin **always** accepts exactly one event per call.

## Configuration
```toml
[[outputs]]
  [outputs.http]
  [outputs.http.serializer]
    type = "avro"

    # avro schema used to encode event data
    schema = '''
    {
      "type": "record",
      "name": "User",
      "fields": [
        {"name": "name", "type": "string"},
        {"name": "age", "type": "int"}
      ]
    }
    '''
```
