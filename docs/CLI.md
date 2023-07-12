# Cli tool

## `pipeline` command

Usage: 
```
neptunus pipeline [--server-address HTTP_ADDRESS] COMMAND [FLAGS]
```

Get help:
```
neptunus pipeline COMMAND -h
```

#### `list`
List all pipelines  
Returns pipelines table
```
id              state   autorun error
--              -----   ------- -----
test.pipeline.1 running true    <nil>
test.pipeline.2 created false   <nil>
```

Flags: **No**.

#### `describe`
Describe pipeline by name (Id)  
Returns pipeline configuration in specified format

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
