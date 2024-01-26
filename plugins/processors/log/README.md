# Log Processor Plugin

The `log` processor writes events into logs at configured level. This plugin requires serializer. If event serialization fails, event will be skipped.

## Configuration
```toml
[[processors]]
  [processors.log]
    # logging level, "debug", "info" or "warn"
    level = "info"
  [processors.log.serializer]
    type = "json"
    data_only = false
```
