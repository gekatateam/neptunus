# Starlark Common Plugin

This plugin provides [Starlark](https://github.com/google/starlark-go/blob/master/doc/spec.md) to Neptunus plugins.

## Builtin types

This plugin defines new type - `event` - as Neptunus event representation in starlark code with methods referenced to [Event API](../../../docs/DATA_MODEL.md):
 - `getId() (id String)` - get event id
 - `setId(key String)` - set event id
 - `getRK() (key String)` - get event routing key
 - `setRK(key String)` - set event routing key
 - `getTimestamp() (t Time)` - get event timestamp
 - `setTimestamp(t Time)` - set event timestamp
 - `setLabel(key String, value String)` - add/overwrite event label
 - `getLabel(key String) (value String|None)` - get label value by key; if label does not exist, **None** returns
 - `delLabel(key String)` - delete label by key
 - `getField(path String) (value String|Bool|Number|Float|Dict|List|Time|Duration|None)` - get field value by path; see [type conversions](../../common/starlark/README.md#type-conversions)
 - `setField(path String, value String|Bool|Number|Float|Dict|List|Time|Duration) (error Error|None)` - set field value by path; see [type conversions](../../common/starlark/README.md#type-conversions)
 - `delField(path String)` - delete field by path
 - `addTag(tag String)` - add tag to event
 - `delTag(tag String)` - delete tag from event
 - `hasTag(tag String) (ok Bool)` - check if event has tag
 - `getErrors() (e List[String])` - get event errors
 - `getUuid() (uuid String)` - get event UUID

Also, you can create a new event using `newEvent(key String) (event Event)` builtin function.

The other new type - `error` - represents Golang **error** type. New error may be created through `error(text String) (error Error)` function. Processing of this type depends on plugins.

## Type conversions
 - Golang nil <-> Starlark None
 - Golang string <-> Starlark String
 - Golang int -> Starlark Int -> Golang int64
 - Golang uint -> Starlark Int -> Golang uint64
 - Golang bool <-> Starlark Bool
 - Golang float -> Starlark Float -> Gloang float64
 - Golang array or slice -> Starlark List -> Gloang slice
 - Golang map[string]T <-> Starlark Dict
 - Golang time.Time <-> starlark Time
 - Golang time.Duration <-> starlark Duration

> [!WARNING]  
> Remember that any event method returns a **value**, not a reference. So if you need to update data, you need to do it directly.

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

> [!WARNING]   
> Modules import is allowed in any script, but import loop checks are not performed.

### Embedded

Three embedded modules are supported:
 - **[time](https://pkg.go.dev/go.starlark.net/lib/time)** - provides time-related constants and functions
 - **[math](https://pkg.go.dev/go.starlark.net/lib/math)** - provides basic constants and mathematical functions
 - **[json](https://pkg.go.dev/go.starlark.net/lib/json)** - utilities for converting Starlark values to/from JSON strings

For import, call the `load()` function, after which a module functions and variables will become available for use via module struct:
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
