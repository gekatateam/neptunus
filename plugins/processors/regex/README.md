# Regex Processor Plugin

The `regex` processor extracts substrings from fields and labels using regular expressions with named captures. Captures from labels are saved as new labels, captures from fields are saved as new fields on the first level of data map. **New labels and fields replace existing once**.

Regular expressions syntax described in [regexp/syntax docs](https://pkg.go.dev/regexp/syntax).

Regex processor only works with string fields, any other types will be ignored. All resulting values are also saved as strings.

## Configuration
```toml
[[processors]]
  # "labels" is a "label name -> regexp" map
  [processors.regex.labels]
    sender = '^(?P<host>.+):(?P<port>[0-9]+)$'
  # "fields" is a "field path -> regexp" map
  [processors.regex.fields]
    message = '(?P<url>http(s)://\S+)'
    # use dots as a path separator to access nested keys
    "log.file" = '/(?P<filename>[^\/]+)$'
```
