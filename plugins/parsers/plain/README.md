# Plain Parser Plugin

The `plain` parser plugin saves the input data in the configured field. This parser always produces one event.

> [!TIP]  
> You can save raw []byte from the input as-is by setting `as_string=false` and `field="."`.

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
