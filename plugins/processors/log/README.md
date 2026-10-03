# Log Processor Plugin

The `log` processor writes events to logs at the configured level. This plugin requires a serializer. If event serialization fails, the event is skipped.

## Configuration
```toml
[[processors]]
  [processors.log]
    # logging level, "debug", "info" or "warn"
    level = "info"

    # if true, plugin drops event after logging
    # may be useful, if you want to log error and remove event from pipeline
    drop_origin = false
  [processors.log.serializer]
    type = "json"
    data_only = false
```
