# Configuration
Neptunus configuration consists of two parts:
 - application configuration for daemon mode
 - pipelines that are run by demon

Each of them can be written in `json`, `yaml` or `toml` format. Pipelines reads at startup from configured storage.

## Daemon config
Here is a full daemon configuration example:
```toml
[common]
  log_level = "info"
  log_format = "logfmt"
  http_port = ":9600"
  [common.log_fields]
    runner = "local"

[pipeline]
  storage = "fs"
  [pipeline.fs]
    directory = ".pipelines"
    extention = "toml"
```

Common section:
 - `log_level` - logging level, accepts `trace`, `debug`, `info`, `warn`, `error` and `fatal`
 - `log_format` - in which format to write logs, accepts `logfmt` and `json`
 - `http_port` - `address:port` binding for internal http server
 - `log_fields` - a map with fields that will be added to every log entry

Processors can be scaled to multiple `lines` - parallel streams - for cases when events are consumed and produced faster than they are transformed in one stream.

> **Important!** Experimentally founded that scaling can reduce performance if processors cumulatively process events faster than outputs send them (because of filling channels buffers). Use it after testing it first.  