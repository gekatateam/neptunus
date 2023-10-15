# Neptunus metrics

## Pipelines common

Metrics that writes by each pipeline

#### Gauge `pipeline_state`
Pipeline state: 1-5 is for Created, Starting, Running, Stopping, Stopped.

Labels:
 - **pipeline** - pipeline Id

#### Gauge `pipeline_processors_lines`
Number of configured processors lines.

Labels:
 - **pipeline** - pipeline Id

## Plugins common

Metrics that writes by each plugin, depends on plugin kind

#### Summary `input_plugin_processed_events`
Events statistic for inputs.

Quantiles: 0.5, 0.9, 0.99, 1.0

Labels:
 - **pipeline** - pipeline Id
 - **plugin** - plugin type
 - **name** - plugin name (alias)
 - **status** - event status, `failed`, `rejected` or `accepted`

#### Summary `filter_plugin_processed_events`
Events statistic for filters.

Quantiles: 0.5, 0.9, 0.99, 1.0

Labels:
 - **pipeline** - pipeline Id
 - **plugin** - plugin type
 - **name** - plugin name (alias)
 - **status** - event status, `failed`, `rejected` or `accepted`

#### Summary `processor_plugin_processed_events`
Events statistic for processors.

Quantiles: 0.5, 0.9, 0.99, 1.0

Labels:
 - **pipeline** - pipeline Id
 - **plugin** - plugin type
 - **name** - plugin name (alias)
 - **status** - event status, `failed`, `rejected` or `accepted`

#### Summary `output_plugin_processed_events`
Events statistic for outputs.

Quantiles: 0.5, 0.9, 0.99, 1.0

Labels:
 - **pipeline** - pipeline Id
 - **plugin** - plugin type
 - **name** - plugin name (alias)
 - **status** - event status, `failed`, `rejected` or `accepted`

#### Summary `parser_plugin_processed_events`
Events statistic for parser.

Quantiles: 0.5, 0.9, 0.99, 1.0

Labels:
 - **pipeline** - pipeline Id
 - **plugin** - plugin type
 - **name** - plugin name (alias)
 - **status** - event status, `failed`, `rejected` or `accepted`

#### Summary `serializer_plugin_processed_events`
Events statistic for serializers.

Quantiles: 0.5, 0.9, 0.99, 1.0

Labels:
 - **pipeline** - pipeline Id
 - **plugin** - plugin type
 - **name** - plugin name (alias)
 - **status** - event status, `failed`, `rejected` or `accepted`

#### Summary `core_plugin_processed_events`
Events statistic for core plugins.

Quantiles: 0.5, 0.9, 0.99, 1.0

Labels:
 - **pipeline** - pipeline Id
 - **plugin** - plugin type
 - **name** - plugin name (alias)
 - **status** - event status, `failed`, `rejected` or `accepted`

## Plugins optional

Optional metrics that plugins may write. Usually, it's protocol-specific metrics.

### HTTP Server

#### Summary `plugin_http_server_requests_seconds`
Incoming http requests stats.

Quantiles: 0.5, 0.9, 0.99

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **uri** - incoming request path
 - **method** - incoming request method
 - **status** - request status

 ### gRPC Server

#### Summary `plugin_grpc_server_calls_seconds`
Handled RPCs stats.

Quantiles: 0.5, 0.9, 0.99

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **procedure** - method full name
 - **type** - RPC type
 - **status** - RPC status

#### Counter `plugin_grpc_server_called_total`
Total number of started RPCs.

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **procedure** - method full name
 - **type** - RPC type

#### Counter `plugin_grpc_server_completed_total`
Total number of completed RPCs.

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **procedure** - method full name
 - **type** - RPC type

#### Counter `plugin_grpc_server_received_messages_total`
Total number of received stream messages.

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **procedure** - method full name
 - **type** - RPC type

#### Counter `plugin_grpc_server_sent_messages_total`
Total number of sent stream messages.

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **procedure** - method full name
 - **type** - RPC type

 ### gRPC Client

#### Counter `plugin_grpc_client_called_total`
Total number of started RPCs.

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **procedure** - method full name
 - **type** - RPC type

#### Counter `plugin_grpc_client_completed_total`
Total number of completed RPCs.

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **procedure** - method full name
 - **type** - RPC type

#### Summary `plugin_grpc_client_calls_seconds`
Handled RPCs stats.

Quantiles: 0.5, 0.9, 0.99

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **procedure** - method full name
 - **type** - RPC type
 - **status** - RPC status

#### Summary `plugin_grpc_client_received_messages_seconds`
Total number of received stream messages.

Quantiles: 0.5, 0.9, 0.99

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **procedure** - method full name
 - **type** - RPC type

#### Summary `plugin_grpc_client_sent_messages_seconds`
Total number of sent stream messages.

Quantiles: 0.5, 0.9, 0.99

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **procedure** - method full name
 - **type** - RPC type

### Kafka Producer

> **Warning**
> Due to the specifics of the [library](https://github.com/segmentio/kafka-go) used, a gauges are reset after being read

#### Counter `plugin_kafka_writer_messages_count`
Number of messages written by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_bytes_count`
Number of bytes written by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_errors_count`
Number of errors occurred during writing

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_write_seconds_sum`
Number of writes performed by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_write_seconds_count`
Total time spent on writes

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_write_seconds_min`
Min time spent on writes

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_write_seconds_avg`
Average time spent on writes

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_write_seconds_max`
Max time spent on writes

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_batch_seconds_count`
Number of batches created by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_batch_seconds_sum`
Total time spent on filling batches

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_seconds_min`
Min time spent on filling batches

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_seconds_avg`
Average time spent on filling batches

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_seconds_max`
Max time spent on filling batches

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_batch_queue_seconds_count`
Number of batches submitted in queue

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_batch_queue_seconds_sum`
Total time spent in queue

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_queue_seconds_min`
Min time spent in queue

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_queue_seconds_avg`
Average time spent in queue

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_queue_seconds_max`
Max time spent in queue

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_size_min`
Min written batch size

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_size_avg`
Average written batch size

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_size_max`
Max written batch size

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID
---
#### Gauge `plugin_kafka_writer_batch_bytes_min`
Min written batch bytes

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_bytes_avg`
Average written batch bytes

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_bytes_max`
Max written batch bytes

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **client_id** - kafka client ID
