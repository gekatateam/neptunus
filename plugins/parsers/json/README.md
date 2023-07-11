# Json Parser Plugin

The `json` parser plugin parses json into events data map.

The result of this plugin depends on passed data:
 - when json object is passed, plugin produces one event with a data map, that was parsed from object
 - when array of objects is passed, plugin produces an event for each object in the array

In any other cases, e.g. array of strings, numbers, arrays, etc, plugin returns an error.

## Configuration
```toml
[[inputs]]
  [inputs.http]
  [inputs.http.parser]
    type = "json"
```
This plugin has no any specific configuration.
