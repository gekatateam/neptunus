# Zstd Compressor Plugin

The `zstd` compressor compress serialization result using [zstd](https://pkg.go.dev/github.com/klauspost/compress/zstd).

# Configuration
```toml
[[outputs]]
  [outputs.http.serializer]
    type = "json"
    compressor = "zstd"

    # compression level, "Fastest", "BestSpeed", 
    # "BetterCompression", "BestCompression"
    zstd_level = "Default"
```
