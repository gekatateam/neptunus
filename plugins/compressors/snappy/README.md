# Snappy Compressor Plugin

The `snappy` compressor compress serialization result using [snappy](https://pkg.go.dev/github.com/golang/snappy).

# Configuration
```toml
[[outputs]]
  [outputs.http.serializer]
    type = "json"

    # this plugin has no any specific configuration
    compressor = "snappy"
```
