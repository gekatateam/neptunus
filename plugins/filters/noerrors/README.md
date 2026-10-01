# Noerrors Filter Plugin

The `noerrors` filter accepts an event only if it has no errors.

## Configuration
```toml
[[processors]]
  [processors.through]
  [processors.through.filters.noerrors]
    reverse = false
```
This plugin has no specific configuration.
