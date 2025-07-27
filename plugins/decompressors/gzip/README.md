# Gsip Decompressor Plugin

The `gzip` decompressor decompress input data defore parsing stage using [gzip](https://pkg.go.dev/compress/gzip).

# Configuration
```toml
[[inputs]]
  [inputs.http.parser]
    type = "json"

    # this plugin has no any specific configuration
    decompressor = "gzip"
```
