# Plain Parser Plugin

The `plain` parser plugin saves passed data as string in configured field. This parser always produce one event.

## Configuration
```toml
[[inputs]]
  [inputs.http]
  [inputs.http.parser]
    type = "plain"
    field = "event"
```
