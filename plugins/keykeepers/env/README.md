# Env Keykeeper Plugin

The `env` keykeeper allows to get configuration keys from environment variables.

## Configuration
```toml
[[keykeepers]]
  [keykeepers.env]
    alias = "envs"

[[inputs]]
  [inputs.kafka]
    group_id = "@{envs:NEPTUNUS_KAFKA_INPUT_CONSUMER_GROUP}"
  [inputs.kafka.sasl]
    mechanism = "none"
  [inputs.kafka.parser]
    type = "json"

```
