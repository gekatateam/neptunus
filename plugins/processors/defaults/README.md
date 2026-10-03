# Defaults Processor Plugin

The `defaults` processor adds configured labels and fields to an event if they do not exist. If adding a field fails, the event is marked as failed, but the fields check loop is not interrupted.

## Configuration

```toml
[[processors]]
  # "labels" is a "label name <- label value" map
  [processors.defaults.labels]
    dc = 'east-01'
  # "fields" is a "field path <- field value" map
  [processors.defaults.fields]
    tested = true
    # use dots as a path separator to access nested keys
    "log.level" = 'info'
```
