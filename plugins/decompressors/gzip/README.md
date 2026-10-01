# Gzip Decompressor Plugin

The `gzip` decompressor decompresses input data before the parsing stage using [gzip](https://pkg.go.dev/compress/gzip).

# Configuration
```toml
[[inputs]]
  [inputs.http.parser]
    type = "json"

    # this plugin has no any specific configuration
    decompressor = "gzip"
```
