# Log Processor Plugin

The `log` processor writes events into logs at configured level. This plugin requires serializer. If event serialization fails, event will be skipped.

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
