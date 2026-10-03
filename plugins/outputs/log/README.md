# Log Output Plugin

The `log` output writes events to logs at the configured level. This plugin requires a serializer.

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
