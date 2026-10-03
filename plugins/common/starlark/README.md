# Starlark Common Plugin

This plugin provides [Starlark](https://github.com/google/starlark-go/blob/master/doc/spec.md) support for Neptunus plugins.

## Builtin types

This plugin defines a new type, `event`, that represents a Neptunus event in Starlark code, with methods described in the [Event API](../../../docs/DATA_MODEL.md):
 - `getId() (id String)` - get event id
 - `setId(key String)` - set event id
 - `getRK() (key String)` - get event routing key
 - `setRK(key String)` - set event routing key
 - `getTimestamp() (t Time)` - get event timestamp
 - `setTimestamp(t Time)` - set event timestamp
 - `setLabel(key String, value String)` - add/overwrite event label
 - `getLabel(key String) (value String|None)` - get the label value by key; if the label does not exist, returns **None**
 - `delLabel(key String)` - delete label by key
 - `getField(path String) (value String|Bool|Number|Float|Dict|List|Time|Duration|None)` - get the field value by path; see [type conversions](../../common/starlark/README.md#type-conversions)
 - `setField(path String, value String|Bool|Number|Float|Dict|List|Time|Duration) (error Error|None)` - set the field value by path; see [type conversions](../../common/starlark/README.md#type-conversions)
 - `delField(path String)` - delete field by path
 - `addTag(tag String)` - add tag to event
 - `delTag(tag String)` - delete tag from event
 - `hasTag(tag String) (ok Bool)` - check if event has tag
 - `getErrors() (e List[String])` - get event errors
 - `delErrors()` - delete all errors from event
 - `getUuid() (uuid String)` - get event UUID
 - `shareTracker(receiver Event)` - share the tracker with another event; **if the receiver already has a tracker, the method panics**

You can also create a new event using the built-in function `newEvent(key String) (event Event)`.

The other new type, `error`, represents the Go **error** type. A new error can be created using the `error(text String) (error Error)` function. Processing of this type depends on the plugin.

You can handle runtime errors using the `handle` function, which accepts a Starlark `Callable`. `handle` returns an `error` or the lambda result if no error occurred:
```python
load("date.star", "date")

def process(event):
    result = handle(lambda: date.parse_weekday(event.getField("weekday")))
    if type(result) == "error":
        print("parsing failed: {}".format(result))
    else:
        event.setField("expected", date.weekday_of(event.getTimestamp) == result)

    return event
```

## Type conversions
 - Golang nil <-> Starlark None
 - Golang string <-> Starlark String
 - Golang int -> Starlark Int -> Golang int64
 - Golang uint -> Starlark Int -> Golang uint64
 - Golang bool <-> Starlark Bool
 - Golang float -> Starlark Float -> Golang float64
 - Golang []any -> Starlark List -> Golang []any
 - Golang map[string]any <-> Starlark Dict
 - Golang time.Time <-> starlark Time
 - Golang time.Duration <-> starlark Duration

> [!WARNING]  
> Remember that every event method returns a **value**, not a reference. If you need to update data, you must do so explicitly.

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
> Modules can be imported from any script, but import loops are not checked.

### Embedded

List of embedded modules:
 - **[time](https://pkg.go.dev/go.starlark.net/lib/time)** - provides time-related constants and functions
 - **[math](https://pkg.go.dev/go.starlark.net/lib/math)** - provides basic constants and mathematical functions
 - **[json](https://pkg.go.dev/go.starlark.net/lib/json)** - utilities for converting Starlark values to/from JSON strings
 - **[yaml](https://github.com/qri-io/starlib/tree/master/encoding/yaml)** - provides functions for working with yaml data
 - **[base64](https://github.com/qri-io/starlib/tree/master/encoding/base64)** - base64 encoding & decoding functions, often used to represent binary as text
 - **[csv](https://github.com/qri-io/starlib/tree/master/encoding/csv)** - reads comma-separated values
 - **[re](https://github.com/qri-io/starlib/tree/master/re)** - provides regular expressions
 - **[fs](../../../pkg/starlarkfs/)** - implements `os.ReadFile` and `os.ReadDir` functions
 - **[date](../../../pkg/starlarkdate/)** - expands `time` module with months and weekdays
 - **[log](log.go)** - provides plugin logger into code

To import a module, call the `load()` function. The module's functions and variables will then become available through the module struct:
```python
load("math.star",   "math")
load("time.star",   "time")
load("date.star",   "date")
load("json.star",   "json")
load("yaml.star",   "yaml")
load("base64.star", "base64")
load("csv.star",    "csv")
load("fs.star",     "fs")
load("log.star",    "log")

print(time.now())
```

### Custom

Custom modules can also be imported. A user module is a script with predefined functions and variables. For a better experience, it is recommended to combine them into one struct:
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
