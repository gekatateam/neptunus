# Configuration

Neptunus configuration files are written in `json`, `yaml`, or `toml` (but we recommend using `toml`, at least for pipelines).

## Daemon

The daemon section configures the Neptunus application and pipeline engine.

You can also use environment variables in the daemon configuration with the `${MY_VAR}` syntax. Please note that replacement occurs before the file is parsed.

The **Common** section is used for low-level settings:
 - **graceful_timeout**: timeout in seconds for graceful shutdown.
 - **log_level**: Logging level, a global setting for the entire application. Accepts `debug`, `info`, `warn`, and `error`.
 - **log_format**: Logging format. Supports `pretty`, `logfmt`, and `json`.
 - **http_port**: Address for the HTTP API server. See more in the [API documentation](API.md).
 - **log_fields**: A map of fields that will be added to each log entry.
 - **log_replaces**: A map of `regexp = replacer` pairs. All matching substrings in a `message` will be replaced. This may help avoid logging sensitive data, such as authorization tokens.

Here is a common part example:
```toml
[common]
  log_level = "info"
  log_format = "logfmt"
  http_port = ":9600"
  graceful_timeout = 15
  [common.log_fields]
    stage = "dev"
    dc = "east-01"
    host = "${HOSTNAME}"
  [common.log_replaces]
    'Bearer \w+' = "<BEARER TOKEN>"
```

The **Runtime** settings may be useful in ephemeral environments, such as Kubernetes with [VPA](https://kubernetes.io/docs/concepts/workloads/autoscaling/#scaling-workloads-vertically), where you cannot directly set your application's resource requests and limits:
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

The **Engine** section is used for pipelines engine settings:
 - **storage**: What kind of storage will be used.
 - **fail_fast**: Whether to fail on startup if any pipeline returns an error.
 - **async_start**: If true, the engine starts all pipelines asynchronously at startup. Otherwise, it starts them sequentially, one after another.

### FS storage

FS storage uses the file system to load, save, and update pipelines:
 - **directory**: Path to the directory where the pipelines files are stored.
 - **extension**: File extension to use. New files will be created with the specified extension, and existing files with a different extension will be ignored.

This is the default storage for the engine:
```toml
[engine]
  storage = "fs"
  fail_fast = false
  [engine.fs]
    directory = ".pipelines"
    extension = "toml"
```

### PostgreSQL storage

PostgreSQL storage uses the configured database as a pipelines source:
 - **instance**: Neptunus instance name. It MUST be unique for each instance using the same database.
 - **dsn**: Connection string. See details [here](https://pkg.go.dev/github.com/jackc/pgx/v4#ConnConfig) (for TLS configuration too). 
 - **username** & **password**: Authentication credentials. These always take precedence over credentials provided in the DSN.
 - **migrate**: Whether the engine should run migration scripts on startup.

This storage provides locking functionality to the engine: each instance acquires a pipeline lock using the instance name and pipeline ID as the key. A pipeline cannot be deleted or updated while it has active locks. All locks associated with a specific instance are removed at startup if **migrate** is `true`.

Minimal example:
```toml
[engine]
  storage = "postgresql"
  [engine.postgresql]
    dsn      = "postgres://localhost:5432/postgres"
    username = "postgres"
    password = "pguser"
    migrate  = true
```

<details>
  <summary>How to manage pipelines with locks:</summary>
  
  If you run Neptunus in Kubernetes or a similar environment and need to manage pipelines without stopping, updating, and restarting your pods, you can create your own event bus for this purpose. Here is an example based on RabbitMQ that shows how to [handle stop/start requests](examples/selfmanage.consume.toml) and [broadcast them to all running engines](examples/selfmanage.process.toml).

  You can use it to stop and start a pipeline on all replicas with a single pseudo-API call to the `selfmanage.consume` HTTP server. However, deployment, update, and deletion operations should still be performed through the main API.
</details>

## Pipeline

A typical pipeline consists of at least one input, at least one output, and optionally, processors. This is how it works:

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
> Configuration examples are shown in the form accepted/returned by the [CLI utility](CLI.md). A form in which the configuration is stored may be different.

Pipeline settings are not directly related to event processing. These parameters are needed by the engine:
 - **id** - Pipeline identifier. Must be unique within a storage.
 - **lines** - Number of parallel streams of pipeline processors. This can be useful in cases where events are consumed and produced faster than they are transformed in a single stream.
 - **run** - Whether the engine should start the pipeline at daemon startup.
 - **buffer** - The buffer size of plugin channels.
 - **consistency** - Pipeline consistency mode (you can ignore it); `soft` by default. Other modes will be added in future releases.
 - **log_level** - Pipeline log level. Overrides the application log level for the specified pipeline and its plugins.

> [!IMPORTANT]  
> Scaling processors can reduce performance if the lines collectively process events faster than the outputs can send them (due to channel buffer overflow). You should test this thoroughly before using it in production.

Settings example:
```toml
[settings]
  id = "test.pipeline.1"
  lines = 5
  run = true
  buffer = 1_000
```

### Vars
This section is intended for general parameters that can be used via [self keykeeper](../plugins/keykeepers/self/):
```toml
[vars]
  max_connections = 10
  log_level = "info"
```

### Plugins

There are three types of first-order plugins:
 - [Input plugins](../plugins/inputs/) consume events from external sources.
 - [Processor plugins](../plugins/processors/) transform events.
 - [Output plugins](../plugins/outputs/) produce events to external systems.

Inputs work independently and send consumed events to the processors stage. If multiple lines are configured, events are distributed among the streams.

In a single line, events move sequentially from processor to processor, according to their order in the configuration. In a multi-line configuration, it may be useful to know which line an event passed through; just add the [line processor](../plugins/processors/line/) to the pipeline.

After the processors stage, events are cloned for each output. For better performance, you can configure multiple identical outputs and filter events by the label from the line processor.

Inputs, processors, and outputs can have [Filter plugins](../plugins/filters/) for event routing. Each plugin can have only one unique filter, and there is no guarantee of the order in which events pass through the filters.

The old way to reverse a filter is to use the `reverse` parameter. If it is `true`, rejected events go to the accept flow, and accepted events go to the reject flow.

The modern way to do it is `not` wrapper:
```toml
[[processors]]
  [processors.through.filters.not.noerrors]
  [processors.through.filters.not.globs]
    labels = { "CATCH_PHRASE" = "*" }
```

Also, you can use any filter twice - with and without wrapper:
```toml
[[processors]]
  [processors.through.filters.not.globs]
    labels = { "CATCH_PHRASE" = "*" }
  [processors.through.filters.globs]
    labels = { "SYSTEM" = "*" }
```

Well, we should say, that in other formats it is a bit ugly:
```yaml
processors:
  - through:
      filters:
        not:
          globs:
            labels:
              CATCH_PHRASE: '*'
          noerrors: {}
        globs:
          labels:
            SYSTEM: '*'
```

The `not` wrapper has another benefit: it correctly handles metrics from the child filter. If that filter accepts an event, the rejected-events counter is incremented, and vice versa.

In the case of inputs and outputs, a rejected event is removed from the pipeline. In the case of processors, a rejected event goes to the next processor instead. Some processors (for example, the [drop processor](../plugins/processors/drop/)) can also drop unnecessary events.

Inputs, processors, outputs and filters may use [Parser plugins](../plugins/parsers/) and [Serializer plugins](../plugins/serializers/). One plugin can have only one parser and one serializer.

[Compressors](../plugins/compressors/) and [Decompressors](../plugins/decompressors/) are used as part of the serializer and parser configuration. A compressor compresses data after serialization, and a decompressor unpacks data before parsing:
<table>
<tr>
<td> Decompressor </td> <td> Compressor </td>
</tr>
<tr>
<td>

```toml
[[inputs]]
  [inputs.http]
    address = ":9200"
  [inputs.http.parser]
    type = "json"
    split_array = true
    decompressor = "gzip"
```

</td>
<td>

```toml
[[outputs]]
  [outputs.http]
    host = "http://localhost:9200"
  [outputs.http.serializer]
    type = "json"
    data_only = true
    compressor = "gzip"
    gzip_level = "DefaultCompression"
```

</td>
</tr>
</table>

A special type of plugins, [Keykeepers](../plugins/keykeepers/), allows you to reference external data in plugin settings using the `@{%keykeeper alias%:%key request%}` pattern:
```toml
[[keykeepers]]
  [keykeepers.env]
    alias = "envs"

[[inputs]]
  [inputs.kafka]
    group_id = "@{envs:NEPTUNUS_KAFKA_INPUT_CONSUMER_GROUP}"
```

The key request format depends on the specific keykeeper.

Keykeepers are initialized before other plugins. You can also use key substitutions in the configuration of other keykeepers if they are declared later:
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

Similar to keykeepers, but for runtime, [Lookups](../plugins/lookups/) work in the background and receive data from external sources at the specified `interval`. You can then retrieve it using the [lookup processor](../plugins/processors/lookup/). This reduces the number of calls to external systems:
```toml
[[lookups]]
  [lookups.sql]
    alias = "settings_table"

[[processors]]
  [processors.lookup]
    lookup = "settings_table"
    [processors.lookup.labels]
      syscodes = "syscodes"
```

### About plugins configuration

First of all, keykeepers, lookups, inputs, processors, and outputs are lists of plugin maps. Here are examples in different formats:
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

This also means that the order of processors depends on their index in the list. One map in a list can contain several different plugins, but in this case their order is random.

An alias can be assigned to each plugin; it will be applied to logs and metrics. Each alias must be unique across the entire pipeline.

You can also override a plugin's log level using the `log_level` parameter. It overrides the pipeline log level (if configured) and the application log level.
