# Converter Processor Plugin

The `converter` processor converts fields and labels from one type to another.

The conversion settings may seem complicated, but if we look at them more closely, they are quite intuitive.

First, any setting is always of the form `%source%:%path%:%layout%`. The layout is used only for time-based sources and targets and may be omitted. If the layout is not set, [time.RFC3339Nano](https://pkg.go.dev/time#pkg-constants) is used. The layout can also be `unix`, `unix_micro`, or `unix_milli` to convert time [to](https://pkg.go.dev/time#Time.Unix)/[from](https://pkg.go.dev/time#Unix) Unix seconds, microseconds, or milliseconds (when converting any value to `unix`, nanoseconds are always zero).

The source depends on path prefix:
 - `label:content-type` for label;
 - `id:path` for event id;
 - `uuid:path` for event UUID;
 - `timestamp:path` for event timestamp;
 - `routingkey:path` for event routing key;
 - `field:log.line` for field.

The target takes from converter configuration. Use:
 - `id` to convert source to event id. In this case, if source is `uuid`, `timestamp` or `routingkey`, path must be any non-empty string (e.g. same as source, like `uuid:uuid`). In other cases, label or field from path will be used;
 - `timestamp` to convert source to event timestamp. If source is `id` or `routingkey`, path must be any non-empty string (e.g. same as source, like `id:id:unix_milli`). In other cases, label or field from path will be used;
 - `routing_key` to convert source to event routing key. If source is `id`, `uuid` or `timestamp`, path must be any non-empty string (e.g. same as source, like `timestamp:timestamp`). In other cases, label or field from path will be used;
 - `label` to convert source to label. Path is always a label name to set. If source is `field`, field from path will be used;
 - `string`, `integer`, `unsigned`, `float`, `boolean`, `time` or `duration` to convert source to concrete field type. Path is always a field path to set. If source is `label` or `field`, label or field from path will be used.

A few limitations:
 - `uuid` can be converted only to `string`, `label`, `id` or `routing_key`;
 - `time` and `timestamp` can be converted only from/to `time`, `id`, `routingkey`, `timestamp`, `string`, `integer` or `unsigned`;
 - `duration` can be converted only from/to `id`, `routingkey`, `string`, `integer` or `unsigned`.

## Configuration
```toml
[[processors]]
  [processors.converter]
    # if true, integer and unsigned overflow 
    # and negative to unsigned convertion errors will be ignored
    ignore_out_of_range = false

    # configured source will be converted to event Id, routing key or timestamp
    id = "label:event-id"
    routing_key = "id:id"
    timestamp = "field:data.timestamp"

    # configured fields will be converted to event labels
    label = [ "field:datagrid.source" ]

    # other types creates or updates fields only
    string = [ "label:content-type" ]
    integer = [ "field:log.line" ]
    unsigned = []
    float = []
    boolean = []
    time = []
    duration = []
```
