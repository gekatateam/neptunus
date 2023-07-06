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

    # if true, parser uses only events data maps
    # otherwise, it uses the whole events
    data_only = true

    # if true, parser ignores serialization errors
    # and skips failed events
    omit_failed = true
```
