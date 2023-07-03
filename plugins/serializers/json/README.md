# Json Serializer Plugin

The `json` serializer plugin converts events into json.

# Configuration
```toml
[[outputs]]
  [outputs.log]
  [outputs.log.serializer]
    type = "json"

    # plugin mode, "jsonl" or "array"
    # in array mode all passed events will be merged into an array of objects
    # in jsonl mode events are combined in jsonl document with new line as a separator
    mode = "jsonl"

    # if it's true, plugin uses only events data maps
    # otherwise, it uses the whole events
    data_only = true

    # if it's true, 
    omit_failed = true
```
