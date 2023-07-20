# TODO
## Common
 - [x] Pipelines manager
 - [ ] Events buffering and batch producing example
 - [ ] Delivery control, event tracing
 - [x] Parser plugins
 - [x] Serializer plugins
 - [ ] Research binary data processing
 - [ ] Hard consisntency units

## Inputs
 - [ ] Kafka
 - [ ] AMQP

## Outputs
With batching and buffering
 - [ ] Kafka
 - [ ] AMQP
 - [ ] Elasticsearch
 - [ ] PostgreSQL

## Filters
 - [x] Glob for labels, fields, routing keys
 - [ ] Comparison operators for numbers

## Processors
 - [x] Regular expressions
 - [ ] Type convertions (strings, numbers, time, duration, bytes, labels, tags, routing key, fields)
 - [ ] Default values for fields
 - ~~[ ] Math operations with numbers~~ Starlark should be used instead
 - [x] Starlark
 - [ ] Correlation

## Pipeline management
 - [ ] Storages:
   - [x] File system
   - [ ] PostgreSQL
   - [ ] Consul KV

 - [x] Pipelines manager
 - [x] REST api over manager
 - [ ] gRPC api over manager
 - [ ] Frontend
