# Noerrors Filter Plugin

The `noerrors` filter accepts event only if it has no errors.

## Configuration
```toml
[[processors]]
  [processors.through]
  [processors.through.filters.noerrors]
    reverse = false
```
This plugin has no any specific configuration.
