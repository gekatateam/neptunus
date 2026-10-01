# Json Parser Plugin

The `json` parser plugin parses JSON into an event data map.

The result of this plugin depends on the input data:
 - when a JSON object is passed, the plugin produces one event
 - when an array is passed:
   - if `split_array` is `true`, each entry is produced as an event
   - if `split_array` is `false`, the plugin produces one event

## Configuration
```toml
[[inputs]]
  [inputs.http]
  [inputs.http.parser]
    type = "json"
    split_array = true

    # "standard" - https://pkg.go.dev/encoding/json#Unmarshal
    # "goccy" - https://pkg.go.dev/github.com/goccy/go-json#Unmarshal
    unmarshaler = "standard"
```
