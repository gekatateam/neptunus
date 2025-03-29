# Self Keykeeper Plugin

The `self` keykeeper allows to get configuration keys from pipeline `settings` and `vars`.

## Configuration
```toml
[settings]
  id = "test.pipeline.cronjob"
  lines = 1
  run = true
  buffer = 1_000

[vars]
  log_level = "info"

[[keykeepers]]
  [keykeepers.self]
    alias = "self"

[[inputs]]
  [inputs.cronjob]
    location = "UTC"
    [[inputs.cronjob.jobs]]
      name = "@{self:settings.id}"
      schedule = "@every 30s"
      force = true

[[processors]]
  [processors.log]
    level = "@{self:vars.log_level}"
    [processors.log.serializer]
      type = "json"
      data_only = false
```
