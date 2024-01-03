# Starlark Processor Plugin
The `starlark` processor uses a [Starlark](../../common/starlark/README.md) script to process events.

The processor uses builtin type - `event` - as Neptunus event representation in starlark code with methods referenced to [Event API](../../../docs/DATA_MODEL.md):
 - `getId() String` - get event id
 - `setId(key String)` - set event id
 - `getRK() String` - get event routing key
 - `setRK(key String)` - set event routing key
 - `addLabel(key String, value String)` - add/overwrite event label
 - `getLabel(key String) value String|None` - get label value by key; if label does not exist, **None** returns
 - `delLabel(key String)` - delete label by key
 - `getField(path String) value String|Bool|Number|Float|Dict|List|None` - get field value by path; see [type conversions](../../common/starlark/README.md#type-conversions)
 - `setField(path String, value String|Bool|Number|Float|Dict|List) error Error|None` - set field value by path; see [type conversions](../../common/starlark/README.md#type-conversions)
 - `delField(path String)` - delete field by path
 - `addTag(tag String)` - add tag to event
 - `delTag(tag String)` - delete tag from event
 - `hasTag(tag String) ok Bool` - check if event has tag
 <!-- - `copy() event Event` - copy event
 - `clone() event Event` - clone event
 - `done()` - mark event as complete

> [!WARNING]  
> If you create new event using `clone()` or `copy()` method and do not return it from script, you MUST mark that event as completed by calling `done()`
> However, this methods are experimental and may be removed or changed in future releases -->

Also, you can create a new event using `newEvent(key String)` builtin function.

The Starlark script must have a `process` function that accepts an event and returns an event, events list, **error** or **None**. If function does not exists it is a compilation error, if it have other signature, it is a runtime error. When **error** returns, an event marked as failed and provided error adds to an event.

Minimalistic example:
```python
def process(event):
    return event
```
<!-- Processor passes a **clone** of an event to script, so event is not changed when an error occurs. -->

## Configuration
```toml
[[processors]]
  [processors.starlark]
    # script file with code
    file = "script.star"

    # starlark code
    # if both, code and file, are set
    # code will be used
    code = '''
def process(event):
    return event
    '''
```
