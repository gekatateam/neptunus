# Glob Filter Plugin

The `glob` filter accepts an event when its routing key, labels, and fields match the configured globs.

All labels must exist and match at least one glob. All fields must exist, be strings, and match at least one glob. Finally, the event's routing key must match at least one glob; otherwise, the event will be rejected.
If a label or field is not mentioned in the configuration, or if `routing_key` is empty, it is not checked.

Glob syntax is similar to [standard wildcards](https://tldp.org/LDP/GNU-Linux-Tools-Summary/html/x11655.htm).

## Configuration
```toml
[[processors]]
  [processors.through]
  [processors.through.filters.glob]
    reverse = false

    # list of patterns, one of which an event routing key must match
    routing_key = [ "http*" ]

    # "labels" is a "label name <- patterns list" map
    [processors.through.filters.glob.labels]
      # list of patterns, one of which an event label must match
      # if label does not exists or not matched any pattern
      # event will be rejected
      sender = [ "*:8765" ]

    # "fields" is a "field path <- patterns list" map
    [processors.through.filters.glob.fields]
      # list of patterns, one of which an event field must match
      # if field does not exists, not a string or not matched any pattern
      # event will be rejected
      message = [ "*docker*", "*podman*" ]
      # use dots as a path separator to access nested keys
      "log.file" = [ "*daemon.json" ]
```
