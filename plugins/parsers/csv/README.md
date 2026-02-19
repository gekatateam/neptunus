# Csv Parser Plugin

The `csv` parser plugin saves CSV data into one or multiple events.

A few words about plugin modes.

In `horizontal` mode plugin returns one event per line. For example, if your CSV data looks like this:
```csv
first_name,last_name,username
"Rob","Pike",rob
Ken,Thompson,ken
"Robert","Griesemer","gri"
```

The events bodies will be:
```json
{"first_name": "Rob",    "last_name": "Pike",      "username": "rob"},
{"first_name": "Ken",    "last_name": "Thompson",  "username": "ken"},
{"first_name": "Robert", "last_name": "Griesemer", "username": "gri"},
```

In other mode, `vertical`, plugin takes `key_column` column as a map key, and returns one event with body as a map of maps:
```json
{
  "rob": {"first_name": "Rob", "last_name": "Pike"},
  "ken": {"first_name": "Ken", "last_name": "Thompson"}
}
```

## Configuration
```toml
[[inputs]]
  [inputs.http.parser]
    type = "csv"

    # plugin mode, "vertical" or "horizontal"
    mode = "horizontal"

    # column name which value will be used as a map key
    # used and required only in "vertical" mode
    key_column = "param_name"

    # if true, entries from first row will be used as a keys in result maps
    # otherwise, each line will be parsed into slice of strings
    # required in "vertical" mode 
    has_header = true

    # if true, a quote may appear in an unquoted field 
    # and a non-doubled quote may appear in a quoted field
    lazy_quotes = false

    # field delimiter, must be a valid rune
    delimeter = ";"

    # ines beginning with the this character without preceding whitespace are ignored
    # must be a valid rune, not set by default
    comment = ""
```
