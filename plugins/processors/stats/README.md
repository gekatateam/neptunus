# Stats Processor Plugin

The `stats` processor calculates counts, sums, averages, minima, maxima, and cumulative histograms with the configured `buckets`, and stores the last field value as a gauge for each configured field, producing it as an event every `period`.

The plugin collects and produces stats for each combination of field name and label values. If an incoming event has no configured field, or if the field cannot be converted to a number, the field stats are not updated.

If `with_labels` is configured and an incoming event has no configured labels, the event is skipped. On the other hand, with `without_labels`, the plugin uses all event labels except the configured ones. **Only one parameter can be configured at the same time.**

Stats are stored as child fields in the `stats` key.

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

> [!TIP]  
> This plugin may write its own [metrics](../../../docs/METRICS.md#internal-caches)

## Configuration
```toml
[[processors]]
  [processors.stats]
    # if true, plugin metrics cache length exposed as metric
    enable_metrics = false

    # plugin mode, "individual" or "shared"
    # in individual mode each plugin collects and produces it's own stats
    # 
    # in shared mode with multiple processors lines
    # each plugin set uses a shared stats cache
    # 
    # in this mode processor sends ALL stats to one random output every period
    mode = "shared"

    # stats collection, producing and reset interval
    # count, sum and gauge are not reset, other stats are set to zero
    # after stats events are produced
    # if configured value less than 1s, it will be set to 1s 
    period = "1m"

    # metrics TTL
    # any metric that has not been observed longer than TTL will be deleted
    # zero means no limit
    metric_ttl = "30m"

    # routing key with which events will be created
    routing_key = "neptunus.generated.metric"

    # labels of incoming events by which metrics will be grouped
    with_labels = [ "::line", "region" ]

    # labels of incoming events which will be ignored
    without_labels = [ "secret_key" ]

    # histogram buckets; each value will be added to outgoing event as `le` label
    # `+Inf` bucket will be added automatically as `math.MaxFloat64`
    # if you don't need histograms, set this parameter to empty list for better performance
    buckets = [ 0.1, 0.3, 0.5, 0.7, 1.0, 2.0, 5.0, 10.0 ]

    # if true, consumed events will be dropped after stats collection
    drop_origin = false

    # "fields" is a "field path <- stats" map
    # plugin expects: "count", "sum", "gauge", "avg", "min", "max", "histogram"
    # any other value or an empty list will cause an error
    [processors.stats.fields]
      "measurements.count" = ["count", "sum", "avg", "histogram"]
      temperature = ["gauge", "max", "min"]
```
