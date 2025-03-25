# Neptunus
 
Neptunus is a data processing engine for consuming, transforming and producing events. Originally conceived as a central unit of a mediation platform, Neptunus may:
 - receive data from a number of different sources, either from message brokers or by acting as a server
 - manage event streams based on filtering rules
 - transform, enrich and create new events
 - deliver events to consumers in various formats and protocols

Neptunus is based on data processing pipelines - a compositions of six types of plugins:
 - [Inputs](plugins/inputs/) consume events from external sources
 - [Processors](plugins/processors/) transform events
 - [Outputs](plugins/outputs/) produce events to external systems
 - [Filters](plugins/filters/) route events in pipeline by conditions
 - [Parsers](plugins/parsers/) convert raw data into events
 - [Serializers](plugins/serializers/) convert events into data formats for external systems

# Configuration
Neptunus configuration has two parts - daemon config and pipelines.

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

### Get help about cli tool usage:
```
neptunus pipeline --help
```

# How to build
This project uses [Taskfile](https://taskfile.dev/) as a build tool. Out-of-the-box there is three OS and two platform combinations: `linux`, `windows`, `darwin` and `amd64`, `arm64`. You can add more in [builds](./Taskfile.build.yaml) and [packs](./Taskfile.pack.yaml) tasks if you need. All tasks should be cross-platform, but you need to know that it tested on Windows 10 and Linux (Ubuntu 22.04) only.

Then, follow simple steps:
1. Install [Taskfile](https://github.com/go-task/task) and [go-licence-detector](https://github.com/elastic/go-licence-detector)
2. Run `task build:{{ OS }}-{{ PLATFORM }}` to build binary
3. Run `task build:notice` to generate NOTICE.txt file
4. Run `task pack:{{ OS }}-{{ PLATFORM }}` to pack your build
5. Run `task build:docker` or `task build:podman` if you need container image
6. Finally, run `task cleanup` to remove build artifacts from file system
