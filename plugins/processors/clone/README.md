# Clone Processor Plugin

The `clone` processor creates clone of each event with new routing key and extra labels (if configured).

## Configuration
```toml
[[processors]]
  [processors.clone]
    # new routing key for an event copy
    routing_key = "rabbit.events.out.2"

    # if true, new event will be created with origin event timestamp
    save_timestamp = false

    # clones count
    count = 1

    # "labels" is a "label name -> label value" map
    # if label does not exists, it will be added
    # if label already exists, it will be overwritten
    [processors.clone.labels]
      shard_key = "1984"
```
