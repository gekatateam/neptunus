# Starlark Processor Plugin
The `starlark` processor uses a [Starlark](https://github.com/google/starlark-go/blob/master/doc/spec.md) script to process events.

The processor defines new builtin type - `event` - as Neptunus event representation in starlark code with methods referenced to [Event api](../../../docs/DATA_MODEL.md):
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

Also, you can create a new event using `newEvent(key String)` builtin function.

The Starlark script must have a `process` function that accepts an event and returns an event, events list, **error** or **None**. If function does not exists it is a compilation error, if it have other signature, it is a runtime error. When **error** returns, an event marked as failed and provided error adds to an event.

Minimalistic example:
```python
def process(event):
    return event
```

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

## Type conversions
 - Golang string <-> Starlark String
 - Golang int -> Starlark Int -> Golang int64
 - Golang uint -> Starlark Int -> Golang uint64
 - Golang bool <-> Starlark Bool
 - Golang float -> starlark Float -> Gloang float64
 - Golang array or slice -> starlark List -> gloang slice
 - Golang map[string]T <-> starlark Dict

Remember that any method returns a **copy** of the data, not a reference. So if you need to update the data, you need to update a routing key, label, field or tag directly.

```python
# bad
def process(event):
    dictField = event.getField("path.to.field")
    dictField["key"] = "new data"
    return event

# good
def process(event):
    dictField = event.getField("path.to.field")
    dictField["key"] = "new data"
    event.setField("path.to.field", dictField)
    return event

```

## Starlark modules

> **Warning!** Modules import is only allowed in the main script, to avoid import loops.

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
