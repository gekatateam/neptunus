# Zstd Decompressor Plugin

The `zstd` decompressor decompresses input data before the parsing stage using [zstd](https://pkg.go.dev/github.com/klauspost/compress/zstd).

# Configuration
```toml
[[inputs]]
  [inputs.http.parser]
    type = "json"

    # this plugin has no any specific configuration
    decompressor = "zstd"
```
