# TODO
## Common
 - [x] Pipelines manager
 - [x] Events buffering and batch producing example - see [grpc output](../plugins/outputs/grpc/grpc.go) and [Batcher](../plugins/common/batcher/batcher.go)
 - [x] Delivery control
 - [ ] Event tracing
 - [x] Parser plugins
 - [x] Serializer plugins
 - [ ] Research binary data processing
 - [ ] Hard consisntency units
 - [x] Path navigation through arrays and slices, e.g. `data.0.field`

## Inputs
 - [x] Kafka
 - [ ] AMQP
 - [x] ~~PostgreSQL~~ Sql should be used instead
 - [x] gRPC stream

## Outputs
With batching and buffering
 - [x] Kafka
 - [ ] AMQP
 - [x] Elasticsearch
 - [x] SQL
 - [x] gRPC stream

## Filters
 - [x] Glob for labels, fields, routing keys
 - [x] ~~Comparison operators for numbers~~ Starlark should be used instead

## Processors
 - [x] Regular expressions
 - [x] Type convertions (strings, numbers, time, duration, bytes, labels, tags, routing key, fields)
 - [x] Default values for fields
 - [x] ~~Math operations with numbers~~ Starlark should be used instead
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
