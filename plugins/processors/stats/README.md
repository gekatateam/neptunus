# Stats Processor Plugin

The `stats` processor calculates count, sum, average, min, max, cumulative histograms with configured `buckets` and stores field last value as gauge for each configured field and produces it as an event every `period`.

Plugin collects and produces stats for each combination of field name and labels values. If incoming event has no any configured label, event will be skipped. If incoming event has no configured field or field cannot be converted to a number, field stats will not updated.

Stats stored as child fields in `stats` key.

This is the format of stats event:
```json
{
  "id": "af002295-7c47-4323-ae5f-f268fad56340",
  "routing_key": "neptunus.generated.metric", # <- configured routing key
  "timestamp": "2023-08-25T22:29:28.9120822+03:00", # <- time of an event creation
  "tags": [],
  "labels": {
    "::line": "3",
    "region": "US/California",
    "::type": "metric", # <- internal label
    "::name": "traffic.now" # <- field name
  },
  "data": { # <- event data
    "stats": { 
      "count": 11,
      "sum": 125,
      "avg": 11.9
    }
  }
}
```

## Configuration
```toml
[[processors]]
  [processors.stats]
    # plugin mode, "individual" or "shared"
    # in individual mode each plugin collects and produces it's own stats
    # 
    # in shared mode with multiple processors lines
    # each plugin set uses a shared stats cache
    # and sends stats events to plugins channels using ROUND ROBIN algorithm
    mode = "shared"

    # stats collection, producing and reset interval
    # count, sum and gauge are not reset, other stats are set to zero
    # after stats events are produced
    # if configured value less than 1s, it will be set to 1s 
    period = "1m"

    # routing key with which events will be created
    routing_key = "neptunus.generated.metric"

    # labels of incoming events by which metrics will be grouped
    labels = [ "::line", "region" ]

    # histogram buckets; each value will be added to outgoing event as `le` label
    # `+Inf` bucket will be added automatically as `math.MaxFloat64`
    # if you don't need histograms, set this parameter to empty list for better performance
    buckets = [ 0.1, 0.3, 0.5, 0.7, 1.0, 2.0, 5.0, 10.0 ]

    # if true, consumed events will be dropped after stats collection
    drop_origin = false

    # "fields" is a "field path -> stats" map
    # plugin expects: "count", "sum", "gauge", "avg", "min", "max", "histogram"
    # any other value or an empty list will cause an error
    [processors.stats.fields]
      "measurements.count" = ["count", "sum", "avg", "histogram"]
      temperature = ["gauge", "max", "min"]
```
