# Converter Processor Plugin

The `converter` processor converts fields and labels from one type to another. The source depends on path prefix - `label:content-type` for label, `field:log.line` or just `log.line` for field. 

Conversion from label to field (and vice versa) creates new one, field convertion replaces origin.

## Configuration
```toml
[[processors]]
  [processors.converter]
    # if true, integer and unsigned overflow 
    # and negative to unsigned convertion errors will be ignored
    ignore_out_of_range = false

    # configured source will be converted to event Id
    id = "label:event-id"

    # configured fields will be converted to event labels
    label = [ "datagrid.source" ]

    # other types creates or updates fields only
    string = [ "label:content-type" ]
    integer = [ "field:log.line" ]
    unsigned = []
    float = []
    boolean = []
```
