# Starlark Filter Plugin
The `starlark` filter uses a [Starlark](../../common/starlark/README.md) script to filter events.

The filter uses builtin type - `event` - as Neptunus **read-only** event representation in starlark code with methods referenced to [Event API](../../../docs/DATA_MODEL.md):
 - `getId() String` - get event id
 - `getRK() String` - get event routing key
 - `getLabel(key String) value String|None` - get label value by key; if label does not exist, **None** returns
 - `getField(path String) value String|Bool|Number|Float|Dict|List|None` - get field value by path; see [type conversions](../../common/starlark/README.md#type-conversions)
 - `hasTag(tag String) ok Bool` - check if event has tag

The Starlark script must have a `filter` function that accepts an event and returns bool or **error**. If function does not exists it is a compilation error, if it have other signature, it is a runtime error. When **error** returns, an event marked as rejected and provided error adds to an event.

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
      code = '''
def filter(event):
    if event.getField("test") < 37:
        return True # accept event
    else:
        return False # reject event
      '''
```
