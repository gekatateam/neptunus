# Plain Parser Plugin

The `plain` parser plugin saves passed data in configured field. This parser always produce one event.

> [!TIP]  
> You can save raw []byte from input as-is using `as_string=false` and `field="."` settings

## Configuration
```toml
[[inputs]]
  [inputs.http]
  [inputs.http.parser]
    type = "plain"

    # if true, []byte will be converted to string
    as_string = true

    # field path to saved content
    field = "event"
```
