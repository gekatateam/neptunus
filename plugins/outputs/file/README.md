# File Output Plugin

The `file` output plugin writes events to a file at the specified path. This plugin requires a serializer.

## Configuration
```toml
[[outputs]]
  [outputs.file]
    # Path to the output file
    path = "/var/log/neptunus/events.log"
    
    # Whether to append to the file if it exists (default: true)
    # If false, the file will be truncated on start
    append = true
    
  [outputs.file.serializer]
    type = "json"
    data_only = true
```

## Notes

- The plugin will automatically create parent directories if they don't exist
- Each event is written on a new line
- Make sure the process has proper permissions to write to the specified path
