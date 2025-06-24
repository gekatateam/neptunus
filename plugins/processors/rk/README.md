# Rk Processor Plugin

The `rk` processor plugin replaces an event routing key with a new one, depending on a current value.

## Configuration
```toml
[[processors]]
  # "mapping" is a "new key <- old keys" map
  # event routing key will be replaced with a new value 
  # if current key matches one in a list
  [processors.rk.mapping]
    "events.topic.0" = [ "/http", "input.queue" ]
    "events.topic.1" = [ "/push" ]
```
