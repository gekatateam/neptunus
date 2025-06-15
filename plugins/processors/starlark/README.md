# Starlark Processor Plugin
The `starlark` processor uses a [Starlark](../../common/starlark/README.md) script to process events.

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

    # script constants, for cases, when you need to provide some data
    # to your generic script
    # each value can be used by it's name in map
    # each value will be converted to Starlark type by rules 
    # described in `Type conversions` paragraph 
    [processors.starlark.constants]
      thenumber = 42
      hostname = "@{env:COMPUTERNAME}"
      table = { a = "a", b = true }
```
