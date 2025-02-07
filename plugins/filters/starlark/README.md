# Starlark Filter Plugin
The `starlark` filter uses a [Starlark](../../common/starlark/README.md) script to filter events.

The filter uses `event`, but with **read-only** methods.

Starlark script must have a `filter` function that accepts an event and returns bool or **error**. If function does not exists it is a compilation error, if it have other signature, it is a runtime error. When **error** returns, provided error added to an event and filter rejects it.

Minimalistic example:
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
```
