# Starlark Processor Plugin
The `starlark` processor the processor uses a [Starlark](https://github.com/google/starlark-go/blob/master/doc/spec.md) script to process events.

The processor defines new builtin type - `event` - as Neptunus event representation in starlark code with methods:
 - `getRK() String` - get event routing key
 - `setRK(key String)` - set event routing key
 - `addLabel(key String, value String)` - add/overwrite event label
 - `getLabel(key String) value String|None` - get label value by key; if label does not exist, **None** returns
 - `delLabel(key String)` - delete label by key
 - `getField(path String) value String|Bool|Number|Float|Dict|List|None` - get field value by path; see [type conversions](#type-conversions)
 - `setField(path String, value String|Bool|Number|Float|Dict|List) error Error|None` - set field value by path; see [type conversions](#type-conversions)
 - `delField(path String)` - delete field by path
 - `addTag(tag String)` - add tag to event
 - `delTag(tag String)` - delete tag from event
 - `hasTag(tag String) ok Bool` - check if event has tag
 - `copy() event Event` - copy event
 - `clone() event Event` - clone event

The Starlark script must have a `process` function that accepts an event and returns an event, events list or **None**. If function does not exists it is a compilation error, if it have other signature, it is a runtime error.

Minimalistic example:
```python
def process(event):
    return event
```

## Type conversions

## Configuration
