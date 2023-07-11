# Glob Filter Plugin

The `glob` filter accepts event when it's routing key, labels and fields mathes configured globs. 

All labels must exist and match any glob, all fields must exist, be a string and match any glob, finally, event routing key must match any glob, otherwise the event will be rejected.  
If label or field not mentioned in configuration, if `routing_key` is empty, it's not checking.

Glob syntax is similar to [standard wildcards](https://tldp.org/LDP/GNU-Linux-Tools-Summary/html/x11655.htm).

## Configuration
```toml
[[processors]]
  [processors.through]
  [processors.through.filters.glob]
    # list of patterns, one of which an event routing key must match
    routing_key = [ "http*" ]

    # "labels" is a "label name -> patterns list" map
    [processors.through.filters.glob.labels]
      # list of patterns, one of which an event label must match
      # if label does not exists or not matched any pattern
      # event will be rejected
      sender = [ "*:8765" ]

    # "fields" is a "field path -> patterns list" map
    [processors.through.filters.glob.filters]
      # list of patterns, one of which an event field must match
      # if field does not exists, not a string or not matched any pattern
      # event will be rejected
      message = [ "*docker*", "*podman*" ]
      # use dots as a path separator to access nested keys
      "log.file" = [ "*daemon.json" ]
```
