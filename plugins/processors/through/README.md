# Through Processor Plugin

The `through` processor passes all events. Yes, that's all. Well, it can sleep, if configured.

## Configuration
```toml
[[processors]]
  [processors.through]
    sleep = "10s"
```
