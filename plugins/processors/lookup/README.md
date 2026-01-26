# Lookup Processor Plugin

The `lookup` processor adds configured labels and fields to an event using data from configured lookup.

There are several places where an error may occur:
 - when converting to a string if target is label;
 - on field set;
 - on lookup call.

If it happens, event will be marked as failed.

## Configuration

```toml
[[lookups]]
  [lookups.http]
    alias = "cloud.token"
    ......

[[processors]]
  [processors.lookup]
    # lookup alias to use
    lookup = "cloud.token"

    # "labels" is a "label name <- lookup key" map
    [processors.lookup.labels]
      # use dots as a path separator to access nested keys
      auth_token = 'response.token'

    # "fields" is a "field path <- lookup key" map
    [processors.lookup.fields]
      claims = 'response.claims'
```
