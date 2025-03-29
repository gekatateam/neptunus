# Configuration

Neptunus configuration files is written using `json`, `yaml` or `toml`.

## Daemon

The daemon part configures Neptunus app and pipelines engine.

You can also use environment variables in daemon config with `${MY_VAR}` syntax. Please note than replacement occurs before file parsing.

**Common** section used for low-level settings:
 - **log_level**: Logging level, global setting for all application, accepts `debug`, `info`, `warn` and `error`.
 - **log_format**: Logging format, supports `pretty`, `logfmt` and `json` formats.
 - **http_port**: Address for HTTP api server. See more in [API doc](API.md).
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
    host = "${HOSTNAME}"
```

**Runtime** settings may help in ephemeral runtimes, like Kubernetes with [VPA](https://kubernetes.io/docs/concepts/workloads/autoscaling/#scaling-workloads-vertically), where you can't directly set your app resources and limits:
 - **gcpercent**: [Garbage collection target percentage](https://pkg.go.dev/runtime/debug#SetGCPercent). Only used if not empty, value must be a percentage, e.g. `25%` or `75%`.
 - **memlimit**: [Soft memory limit](https://pkg.go.dev/runtime/debug#SetMemoryLimit). Only used if not empty, value can be a percentage from available memory (e.g. `25%` or `75%`) or an absolute (for example, `1GiB` or `512MiB`).
 - **maxthreads**: [The maximum number of operating system threads that the Go program can use](https://pkg.go.dev/runtime/debug#SetMaxThreads). Only used if greater than zero, integer value.
 - **maxprocs**: [The maximum number of CPUs that can be executing simultaneously](https://pkg.go.dev/runtime#GOMAXPROCS). Only used if greater than zero, integer value.

```toml
[runtime]
  gcpercent = "50%"
  memlimit = "70%"
  maxthreads = 10000
  maxprocs = 4
```

**Engine** section used for pipelines engine settings:
 - **storage**: What kind of storage should be used.
 - **fail_fast**: Fail on startup, if any pipeline returns error.

### FS storage

FS storage uses the file system to load, save and update pipelines:
 - **directory**: Path to the directory where the pipelines files are stored.
 - **extention**: Files with which extension to use. New files will be created with the specified extension, existing files with a different extension will be ignored.

This is a default storage for the engine:
```toml
[engine]
  storage = "fs"
  fail_fast = false
  [engine.fs]
    directory = ".pipelines"
    extention = "toml"
```

## Pipeline

Typical pipeline consists of at least one input, at least one output and, not necessarily, processors. This is how it works:

<table>
<tr>
<td> Common </td> <td> Input </td> <td> Processor </td> <td> Output </td>
</tr>
<tr>
<td>

```
           processors line            
         ┌─────┬───────────┐          
 ┌───┐   │┌───┐|┌───┐ ┌───┐│   ┌────┐ 
 |>in├┐ ┌┼┤pr1├┼┤pr2├─┤pr3├┼┐ ┌┤out>│ 
 └───┘| ││└───┘|└───┘ └───┘│| │└────┘ 
 ┌───┐| │├┬───┬┼┬───┬─┬───┬┘| │┌────┐ 
 |>in├┼─┼┼┤pr1├┼┤pr2├─┤pr3├─┼─┼┤out>│ 
 └───┘| │|└───┘|└───┘ └───┘ | │└────┘ 
 ┌───┐| │|┌───┐|┌───┐ ┌───┐ | │┌────┐ 
 |>in├┘ └┼┤pr1├┼┤pr2├─┤pr3├─┘ └┤out>│ 
 └───┘   |└───┘|└───┘ └───┘    └────┘ 
         └─────┘ 
      processors set
```

</td>
<td>

```
 ┌────────────────┐
 |┌───┐ ┌───┐ rej |
 ||>in├─┤ f ├┬──Θ |
 |└───┘ └─┬┬┴┴─┐  |
 |        └┤ f ├──┼>
 |         └───┘  |
 └────────────────┘
```

</td>
<td>

```
 ┌────────────────┐
 |┌───┐ rejected  |
>┼┤ f ├┬─────────┐|
 |└─┬┬┴┴─┐ ┌────┐||
 |  └┤ f ├─┤proc├┴┼>
 |   └───┘ └────┘ |
 └────────────────┘
```

</td>
<td>

```
 ┌────────────────┐
 |┌───┐ rejected  |
>┼┤ f ├┬────────Θ |
 |└─┬┬┴┴─┐ ┌────┐ |
 |  └┤ f ├─┤out>| |
 |   └───┘ └────┘ |
 └────────────────┘
```

</td>
</tr>
</table>

### Settings

> [!NOTE]  
> Configuration examples are shown in the form accepted/returned by the [CLI utility](CLI.md). A form in which the configuration is stored depends on a storage.

Pipeline settings are not directly related to event processing, these parameters are needed for the engine:
 - **id** - Pipeline identificator. Must be unique within a storage.
 - **lines** - Number of parallel streams of pipeline processors. It can be useful in cases, when events are consumed and produced faster than they are transformed in one stream. 
 - **run** - Should engine starts pipeline at daemon startup.
 - **buffer** - Buffer size of channels connecting a plugins.
 - **consistency** - Pipeline consistency mode; `soft` by default, `hard` mode will be added in future releases;

> [!IMPORTANT]
> Processors scaling can reduce performance if the lines cumulatively process events faster than outputs send them (because of channels buffers overflow). You should test it first before use it in production.  

Settings example:
```toml
[settings]
  id = "test.pipeline.1"
  lines = 5
  run = true
  buffer = 1_000
```

### Vars
This section is intended for storing general parameters that can be used via [`self` keykeeper](../plugins/core/self/). Well, `settings` block is also available:
```toml
[vars]
  max_connections = 10
  log_level = "info"
```

### Plugins

There are three types of first-order plugins:
 - [Input plugins](../plugins/inputs/) consume events from external sources.
 - [Processor plugins](plugins/processors/) transform events.
 - [Output plugins](plugins/outputs/) produce events to external systems.

Inputs works independently and send consumed events to the processors stage. If multiple lines configured, events are distributed between streams.

In one line events move sequentially, from processor to processor, according to an order in configuration. In multi-line configuration, it may be useful to understand which line an event passed through - just add [line processor](../plugins/processors/line/) in pipeline.

After processors, events are cloned to each output. For better performance, you can configure multiple identical outputs and filter events by label from line processor.

Inputs, processors and outputs can have [Filter plugins](../plugins/filters/) for events routing. Each plugin can have only one unique filter, and there is no guarantee of the order in which events pass through the filters. Each filter can be reversed using `reverse` parameter. If it's `true`, rejected events goes to accept flow, and accepted events goes to reject.

In inputs and outputs case, if any filter rejects event, the event is dropped from pipeline. In processors case, otherwise, rejected event going to a next processor. Some processors (for example, [drop processor](../plugins/processors/drop/)) also can drop unnecessary events.

Inputs, processors, outputs and filters may use [Parser plugins](../plugins/parsers/) and [Serializer plugins](../plugins/serializers/) (it depends on plugin). One plugin can have only one parser and one serializer.

A special plugins, [Keykeepers](../plugins/keykeepers/), allows you to reference external data in plugins settings using `@{%keykeeper alias%:%key request%}` pattern:
```toml
[[keykeepers]]
  [keykeepers.env]
    alias = "envs"

[[inputs]]
  [inputs.kafka]
    group_id = "@{envs:NEPTUNUS_KAFKA_INPUT_CONSUMER_GROUP}"
```

Key request format depends on concrete keykeeper used.

Keykeepers are initialized before other plugins. Also, you can use key substitutions in other keykeepers configuration if they are declared after:
```toml
[[keykeepers]]
  [keykeepers.env]
    alias = "envs"

[[keykeepers]]
  [keykeepers.vault]
    alias = "vault"
    address = "https://vault.local:443"
    [keykeepers.vault.approle]
      role_id = "@{envs:HASHICORP_VAULT_ROLE_ID}"
      secret_id = "@{envs:HASHICORP_VAULT_SECRET_ID}"
```

### About plugins configuration

First of all, keykeepers, inputs, processors and outputs is a list of plugins map. Here is an example in different formats:
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

[[processors]]
  [processors.line]

[[processors]]
  [processors.log]
    level = "warn"
  [processors.log.serializer]
    type = "json"
    data_only = false
  [processors.log.filters.glob]
    routing_key = [ "*http.*" ]

[[outputs]]
  [outputs.log]
    level = "info"
  [outputs.log.serializer]
    type = "json"
    data_only = true
    mode = "array"
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

processors:
  - line: {}
  - log:
      level: warn
      serializer:
        type: json
        data_only: false
      filters:
        glob:
          routing_key:
            - '*http.*'

outputs:
  - log:
      level: info
      serializer:
        type: json
        data_only: true
        mode: array

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
    }
  ],
  "processors": [
    {
      "line": {}
    },
    {
      "log": {
        "level": "warn",
        "serializer": {
          "type": "json",
          "data_only": false
        },
        "filters": {
          "glob": {
            "routing_key": [
              "*http.*"
            ]
          }
        }
      }
    }
  ],
  "outputs": [
    {
      "log": {
        "level": "info",
        "serializer": {
          "type": "json",
          "data_only": true,
          "mode": "array"
        }
      }
    }
  ]
}
```

</td>
</tr>
</table>

This also means that the order of processors depends on their index in a list. One map in a list can contain several different plugins, but in this case their order will be random.

An alias can be assigned to each plugin - this will affect logs and metrics. Each alias must be unique.
