# Starlark Common Plugin

This plugin provides [Starlark](https://github.com/google/starlark-go/blob/master/doc/spec.md) in Neptunus plugins ecosystem.

## Type conversions
 - Golang string <-> Starlark String
 - Golang int -> Starlark Int -> Golang int64
 - Golang uint -> Starlark Int -> Golang uint64
 - Golang bool <-> Starlark Bool
 - Golang float -> Starlark Float -> Gloang float64
 - Golang array or slice -> Starlark List -> Gloang slice
 - Golang map[string]T <-> Starlark Dict

> [!WARNING]  
> Remember that any method returns a **copy** of the data, not a reference. So if you need to update the data, you need to update a routing key, label, field or tag directly.

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
