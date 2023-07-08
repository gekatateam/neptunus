# Configuration

Neptunus configuration files is written using `json`, `yaml` or `toml`.

## Daemon

The daemon part configures Neptunus engine and pipelines storage.

Common section used for low-level settings:
 - **log_level**: Logging level, global setting for all application, accepts `trace`, `debug`, `info`, `warn`, `error` and `fatal`.
 - **log_format**: Logging format, supports `logfmt` and `json` formats.
 - **http_port**: Address for HTTP api server. See more in [api documentation](API.md).
 - **log_fields**: A map of fields, that will be added to each log entry.

Here is a common part example:
```toml
[common]
  log_level = "info"
  log_format = "logfmt"
  http_port = ":9600"
  [common.log_fields]
    stage = "dev"
    dc = "east-01"
```


Pipeline section used for pipelines engine settings:
 - **storage**: What kind of storage should be used.

### FS storage

FS storage uses the file system to load, save and update pipelines:
 - **directory**: Path to the directory where the pipelines files are stored.
 - **extention**: Files with which extension to use. New files will be created with the specified extension. Files with a different extension will be ignored. 

This is a default storage for engine:
```toml
[pipeline]
  storage = "fs"
  [pipeline.fs]
    directory = ".pipelines"
    extention = "toml"
```























<!--
Processors can be scaled to multiple `lines` - parallel streams - for cases when events are consumed and produced faster than they are transformed in one stream.

> **Important!** Experimentally founded that scaling can reduce performance if processors cumulatively process events faster than outputs send them (because of filling channels buffers). Use it after testing it first.  
-->
