# Cli tool

## `pipeline` command

Usage: 
```
neptunus pipeline [--server-address http://localhost:9600] [--request-timeout 10s] COMMAND [FLAGS]
```

Get help:
```
neptunus pipeline COMMAND -h
```

#### `list`
List all pipelines  
Returns pipelines table
```
id                        autorun state   last_error
--                        ------- -----   ----------
realtime.orderbook.FUTURE true    running <nil>
test.pipeline.beats       false   created <nil>
```

Flags:
 - *--format* - list format (plain, json, yaml supported)

#### `describe`
Describe pipeline by name (Id)  
Returns pipeline configuration in specified format  
Additional info included in `runtime` block - current state and last error

Flags:
 - **--name** - Pipeline name
 - *--format* - pipeline printing format (json, toml, yaml supported)

#### `deploy`
Deploy new pipeline from file (json, toml, yaml supported)

Flags:
 - **--file** - Path to file with pipeline configuration

#### `update`
Update existing pipeline from file (json, toml, yaml supported)

Flags:
 - **--file** - Path to file with pipeline configuration

#### `delete`
Delete pipeline by name (Id)

Flags:
 - **--name** - Pipeline name

#### `start`
Start pipeline by name (Id)

Flags:
 - **--name** - Pipeline name

#### `stop`
Stop pipeline by name (Id)

Flags:
 - **--name** - Pipeline name
