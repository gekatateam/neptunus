# Snappy Decompressor Plugin

The `snappy` decompressor decompress input data defore parsing stage using [snappy](https://pkg.go.dev/github.com/golang/snappy).

# Configuration
```toml
[[inputs]]
  [inputs.http.parser]
    type = "json"

    # this plugin has no any specific configuration
    decompressor = "snappy"
```
