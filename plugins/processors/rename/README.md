# Rename Processor Plugin

The `rename` processor can be used to rename or copy labels and fields.

## Configuration
```toml
[[processors]]
  [processors.rename]
    # if true, the original labels and fields will be deleted
    # after a successful copy to a new path
    delete_origin = false

    # "labels" is a "new name <- old name" map
    # if a label with the "old name" exists,
    # its value will be copied to the "new name"
    [processors.rename.labels]
      "x-stage"  = "stage"
      "x-region" = "region"

    # "fields" is a "new path <- old path" map
    # if a field at the "old path" exists,
    # its value will be copied to the "new path"
    [processors.rename.fields]
      "result" = "status"
```
