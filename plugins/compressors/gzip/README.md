# Gzip Compressor Plugin

The `gzip` compressor compress serialization result using [gzip](https://pkg.go.dev/compress/gzip).

# Configuration
```toml
[[outputs]]
  [outputs.http.serializer]
    type = "json"
    compressor = "gzip"

    # compression level, "NoCompression", "BestSpeed", 
    # "BestCompression", "DefaultCompression", "HuffmanOnly"
    gzip_level = "DefaultCompression"
```
