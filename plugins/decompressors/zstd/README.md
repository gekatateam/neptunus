# Zstd Decompressor Plugin

The `zstd` decompressor decompress input data defore parsing stage using [sztd](https://pkg.go.dev/github.com/klauspost/compress/zstd).

# Configuration
```toml
[[inputs]]
  [inputs.http.parser]
    type = "json"

    # this plugin has no any specific configuration
    decompressor = "zstd"
```
