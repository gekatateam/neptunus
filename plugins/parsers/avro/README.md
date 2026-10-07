# Avro Parser Plugin

The `avro` parser plugin decodes Avro binary data into event data. The Avro schema must be provided in the parser configuration. Only `record`, `map`, and `array` schemas are supported at the top level.

The result depends on the top-level schema:
 - when a record or map is passed, the plugin produces one event
 - when an array is passed:
   - if `split_array` is `true`, each entry is produced as an event
   - if `split_array` is `false`, the entire array is produced as one event

## Configuration
```toml
[[inputs]]
  [inputs.http]
  [inputs.http.parser]
    type = "avro"

    
    split_array = true


    schema = '''
    {
      "type": "array",
      "items": {
        "type": "record",
        "name": "User",
        "fields": [
          {"name": "name", "type": "string"},
          {"name": "age", "type": "int"}
        ]
      }
    }
    '''
```
