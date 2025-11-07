# SwitchCase Processor Plugin

The `switchcase` processor allows the mapping of labels and fields between it's values.

Just like `switch-case` in Go:
```go
field, _ := event.GetField("status.code")
switch field {
case "0", "1", "2":
    event.SetField("status.code", "success")
case "3", "4", "5":
    event.SetField("status.code", "faled")
}
```

> [!WARNING]  
> Due to runtime panic-safety this plugin works only with strings. The [Converter](../converter/) processor can help you here.

## Configuration
```toml
[[processors]]
  # "labels" is a "label name" <- "mappings" dict
  [processors.switchcase.labels.region]

    # each mapping is a "new value" <- "old values" dict
    # if label exists and it value in "old values" list, 
    # it will be replaced with "new value"
    "west" = [ "w-1", "w-2", "w-3" ]
    "east" = [ "e-1", "e-2", "e-3" ]
  [processors.switchcase.labels.stage]
    "test" = [ "test", "feature", "integration", "load" ]

  # "fields" is a "field path" <- "mappings" dict
  # it works just like "labels"
  # if field exists, it is a string and it value in "old values" list, 
  # it will be replaced with "new value"
  [processors.switchcase.fields."status.code"]
    "success" = [ "0", "1", "2" ]
    "failed"  = [ "3", "4", "5" ]
```
