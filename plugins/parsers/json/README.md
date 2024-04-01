# Json Parser Plugin

The `json` parser plugin parses json into events data map.

The result of this plugin depends on passed data:
 - when json object is passed, plugin produces one event
 - when array passed:
   - if `split_array` is `true`, each entry will be produced as an event
   - if `split_array` is `false`, plugin produces one event

## Configuration
```toml
[[inputs]]
  [inputs.http]
  [inputs.http.parser]
    type = "json"
    split_array = true
```
