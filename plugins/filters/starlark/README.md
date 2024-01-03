# Starlark Filter Plugin
The `starlark` filter uses a [Starlark](https://github.com/google/starlark-go/blob/master/doc/spec.md) script to filter events.

The filter defines new builtin type - `event` - as Neptunus **read-only** event representation in starlark code with methods referenced to [Event API](../../../docs/DATA_MODEL.md):
 - `getId() String` - get event id
 - `getRK() String` - get event routing key
 - `getLabel(key String) value String|None` - get label value by key; if label does not exist, **None** returns
 - `getField(path String) value String|Bool|Number|Float|Dict|List|None` - get field value by path; see [type conversions](#type-conversions)
 - `hasTag(tag String) ok Bool` - check if event has tag

The Starlark script must have a `filter` function that accepts an event and bool or **error**. If function does not exists it is a compilation error, if it have other signature, it is a runtime error. When **error** returns, an event marked as rejected and provided error adds to an event.

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
        return True
    else:
        return False
      '''
```

## Starlark modules

> [!WARNING]   
> Modules import is allowed in any script, but import loop checks are not performed.

### Embedded

Three embedded modules are supported:
 - **[time](https://pkg.go.dev/go.starlark.net/lib/time)** - provides time-related constants and functions
 - **[math](https://pkg.go.dev/go.starlark.net/lib/math)** - provides basic constants and mathematical functions
 - **[json](https://pkg.go.dev/go.starlark.net/lib/json)** - utilities for converting Starlark values to/from JSON strings

To import, call the `load()` function, after which the module functions and variables will become available for use via the module struct:
```python
load("math.star", "math")
load("time.star", "time")
load("json.star", "json")

print(time.now())
```

### Custom

Import of custom modules is also supported. A user module is just a script with predefined functions and variables, but for better experience it is recommended to combine them into one struct:
```python
# myModule.star
helloMessage = "hello from module"

def hello():
    print(helloMessage)

myModule = struct(
    hello = hello
    helloMessage = helloMessage
)
```

Then use of the module will not differ from the built-in ones:
```python
# process.star
load("myModule.star", "myModule")

myModule.hello() # prints "hello from module"
```
