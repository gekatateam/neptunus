# Not Filter Plugin

The `not` filter negates the result of the underlying filter. If the underlying filter accepts an event, the `not` filter will reject it, and vice versa.

## Configuration
```toml
[[processors]]
  [processors.through.filters.not.globs]
    labels = { "CATCH_PHRASE" = "*" }
```
This plugin has no specific configuration and cannot be used without child filters.
