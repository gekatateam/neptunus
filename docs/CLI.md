# Cli tool

## `pipeline` command

Usage: 
```
neptunus pipeline [OPTIONS] COMMAND [FLAGS]
```

Options:
 - *--server-address*/*-s* - daemon http api server address; if sheme is HTTPS, tls 
transport will be used (default: "http://localhost:9600")
 - *--request-timeout*/*-t* - api call timeout (default: 10s)
 - *--tls-key-file*/*--tk* - path to TLS key file
 - *--tls-cert-file*/*--tc* - path to TLS certificate file
 - *--tls-ca-file*/*--ta* - path to TLS CA file
 - *--tls-skip-verify*/*--ts* - skip TLS certificate verification (default: false)     
 - *--header*/*-H* - custom header to add to request (can be repeated), format: Key:Value

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
 - **--name**/**--id** - Pipeline name
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
 - **--name**/**--id** - Pipeline name

#### `start`
Start pipeline by name (Id)

Flags:
 - **--name**/**--id** - Pipeline name

#### `stop`
Stop pipeline by name (Id)

Flags:
 - **--name**/**--id** - Pipeline name
