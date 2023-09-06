# Log Output Plugin

The `log` output writes events into logs at configured level. This plugin requires serializer.

## Configuration
```toml
[[outputs]]
  [outputs.log]
    # logging level, "debug", "info" or "warn"
    level = "info"
  [outputs.log.serializer]
    type = "json"
    data_only = true
```
