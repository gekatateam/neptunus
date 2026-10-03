# Starlark Filter Plugin
The `starlark` filter uses a [Starlark](../../common/starlark/README.md) script to filter events.

The filter uses `event`, but with **read-only** methods.

The Starlark script must have a `filter` function that accepts an event and returns a boolean or an **error**. If the function does not exist, it is a compilation error; if it has another signature, it is a runtime error. When an **error** is returned, the error is added to the event and the filter rejects it.

Minimal example:
```python
def filter(event):
    return True
```

## Configuration
```toml
[[outputs]]
  [outputs.log]
    level = "info"
    [outputs.log.serializer]
      type = "json"
      data_only = false
    [outputs.log.filters.starlark]
      reverse = false

      # script file with code
      file = "script.star"

      # starlark code
      # if both, code and file, are set
      # code will be used
      code = '''
def filter(event):
    if event.getField("test") < 37:
        return True # accept event
    else:
        return False # reject event
      '''

      # script constants, for cases, when you need to provide some data
      # to your generic script
      # each value can be used by it's name in map
      # each value will be converted to Starlark type by rules 
      # described in `Type conversions` paragraph 
      [outputs.log.filters.starlark.constants]
        thenumber = 42
        hostname = "@{env:COMPUTERNAME}"
        table = { a = "a", b = true }
```
