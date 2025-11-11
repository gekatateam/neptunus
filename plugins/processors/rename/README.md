# Rename Processor Plugin

The `rename` processor can be used to rename (or copy) labels and fields.

## Configuration
```toml
[[processors]]
  [processors.rename]
    # if true, origin labels and fields will be deleted 
    # after successfull copy to a new path
    delete_origin = false

    # "labels" is a "new name <- old name" map
    # if label with "old name" exists,
    # it value will be copied to a "new name"
    [processors.rename.labels]
      "x-stage"  = "stage"
      "x-region" = "region"

    # "fields" is a "new path <- old path" map
    # if field on "old path" exists
    # it value will be copied to a "new path"
    [processors.rename.fields]
      "result" = "status"
```
