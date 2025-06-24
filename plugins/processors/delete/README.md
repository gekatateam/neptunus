# Delete Processor Plugin

The `delete` processor deletes events labels and fields.

## Configuration
```toml
[[processors]]
  [processors.delete]

    # list of labels to delete
    labels = [ "sender", "exchange" ]

    # list of fields to delete
    fields = [ "log.source", "fqdn" ]
```
