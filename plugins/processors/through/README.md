# Through Processor Plugin

The `through` processor passes all events. That's all. Well, it can sleep if configured.

## Configuration
```toml
[[processors]]
  [processors.through]
    sleep = "10s"
```
