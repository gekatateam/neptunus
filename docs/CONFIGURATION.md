# Configuration

Neptunus configuration files is written using `json`, `yaml` or `toml`.

## Daemon

The daemon part configures Neptunus app and pipelines engine.

**Common** section used for low-level settings:
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

**Engine** section used for pipelines engine settings:
 - **storage**: What kind of storage should be used.

### FS storage

FS storage uses the file system to load, save and update pipelines:
 - **directory**: Path to the directory where the pipelines files are stored.
 - **extention**: Files with which extension to use. New files will be created with the specified extension, existing files with a different extension will be ignored.

This is a default storage for the engine:
```toml
[engine]
  storage = "fs"
  [engine.fs]
    directory = ".pipelines"
    extention = "toml"
```

## Pipeline

Typical pipeline consists of at least one input, at least one output and, not necessarily, processors. It also have some personal settings.

### Settings

Pipeline settings are not directly related to event processing, these parameters are needed for the engine:
 - **id** - Pipeline identificator. Must be unique within a storage.
 - **lines** - Number of parallel streams of pipeline processors. It can be useful in cases, when events are consumed and produced faster than they are transformed in one stream. 
 - **run** - Should engine starts pipeline at daemon startup.
 - **buffer** - Buffer size of channels connecting a plugins.

> **Important!** It has been experimentally found that processors scaling can reduce performance if the lines cumulatively process events faster than outputs send them (because of filling channels buffers). You should use scaling after preliminary testing.  

Settings example:
```toml
[settings]
  id = "test.pipeline.1"
  lines = 5
  run = true
  buffer = 1_000
```

### About plugins

First of all, inputs, processors and outputs is a list of plugins map. Here is an example in different formats:
<table>
<tr>
<td> Toml </td> <td> Yaml </td> <td> Json </td>
</tr>
<tr>
<td>

```toml
[[inputs]]
  [inputs.httpl]
    address = ":9200"
    max_connections = 10
  [inputs.httpl.parser]
    type = "json"

[[inputs]]
  [inputs.httpl]
    address = ":9400"
  [inputs.httpl.parser]
    type = "json"
```

</td>
<td>

```yaml
inputs:
  - httpl:
      address: ':9200'
      max_connections: 10
      parser:
        type: json
  - httpl:
      address: ':9400'
      parser:
        type: json
```

</td>
<td>

```json
{
  "inputs": [
    {
      "httpl": {
        "address": ":9200",
        "max_connections": 10,
        "parser": {
          "type": "json"
        }
      }
    },
    {
      "httpl": {
        "address": ":9400",
        "parser": {
          "type": "json"
        }
      }
    }
  ]
}
```

</td>
</tr>
</table>

This also means that the order of the processors depends on their index in a list. One map in a list can contain several different plugins, but in this case their order will be random.

### Inputs



