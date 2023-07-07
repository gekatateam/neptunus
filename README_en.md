# Neptunus

Neptunus is a data processing engine for consuming, transforming and producing events. Neptunus based on data processing pipelines - a compositions of six types of plugins:
 - [inputs](plugins/inputs/) consume events from external sources
 - [processors](plugins/processors/) transform events
 - [outputs](plugins/outputs/) produce events to external systems
 - [filters](plugins/filters/) route events in pipeline by conditions
 - [parsers](plugins/parsers/) convert raw data into events
 - [serializers](plugins/serializers/) convert events into data formats for sending to external systems

Typical pipeline consists of at least one input, at least one output and, not necessarily, processors.

Processors can be scaled to multiple `lines` - parallel streams - for cases when events are consumed and produced faster than they are transformed in one stream.

> **Important!** Experimentally founded that scaling can reduce performance if processors cumulatively process events faster than outputs send them (because of filling channels buffers). Use it after testing it first.  

# Getting Started
## Get help:
```
neptunus --help
```

## Run daemon
```
neptunus run --config config.toml
```

## Test pipelines configuration
```
neptunus test --config config.toml
```

## Get help about cli usage
```
neptunus pipeline --help
```
