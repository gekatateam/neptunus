# Neptunus
 
Neptunus is a data processing engine for consuming, transforming, and producing events. Originally conceived as a central unit of a mediation platform, Neptunus can:
 - receive data from a number of different sources, either from message brokers or by acting as a server,
 - manage event streams based on filtering rules,
 - transform, enrich, and create new events,
 - and deliver events to consumers in various formats and protocols.

It can also [collect](plugins/processors/stats) and [write](plugins/outputs/promremote) metrics directly related to your processes.

Neptunus is based on data processing pipelines - compositions of plugins:
 - [Inputs](plugins/inputs/) consume events from external sources
 - [Processors](plugins/processors/) transform events
 - [Outputs](plugins/outputs/) produce events for external systems
 - [Filters](plugins/filters/) route events in a pipeline based on conditions
 - [Parsers](plugins/parsers/) convert raw data into events
 - [Serializers](plugins/serializers/) convert events into formats for external systems
 - [Compressors](plugins/compressors/) compress serialized events
 - [Decompressors](plugins/decompressors/) decompress raw bytes before parsing
 - [Lookups](plugins/lookups/) retrieve data in the background

# Configuration
Neptunus configuration has two parts: the daemon configuration and pipelines.

See more in our [documentation](docs/CONFIGURATION.md).

# Getting Started
### Get help:
```
neptunus --help
```

### Run daemon:
```
neptunus run --config config.toml
```

### Test pipelines configuration:
```
neptunus test --config config.toml
```

### Get help about CLI tool usage:
```
neptunus pipeline --help
```

# How to build
This project uses [Taskfile](https://taskfile.dev/) as a build tool. Out of the box, there are three operating systems and two platforms: `linux`, `windows`, `darwin`, `amd64`, and `arm64`. You can add more in the [builds](./Taskfile.build.yaml) and [packs](./Taskfile.pack.yaml) tasks if needed. All tasks should be cross-platform, but note that they are tested only on Windows 10, Linux (Ubuntu 22.04), and macOS 26.

Then follow these simple steps:
1. Install [Taskfile](https://github.com/go-task/task) and [go-licence-detector](https://github.com/elastic/go-licence-detector)
2. Run `task build:{{ OS }}-{{ PLATFORM }}` to build the binary
3. Run `task build:notice` to generate the NOTICE.txt file
4. Run `task pack:{{ OS }}-{{ PLATFORM }}` to package your build
5. Run `task build:docker` or `task build:podman` if you need a container image
6. Finally, run `task cleanup` to remove build artifacts from the file system
