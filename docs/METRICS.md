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
> Due to the specifics of the [library](https://github.com/segmentio/kafka-go) used, producer metrics updates every 15 sec, not on read

#### Counter `plugin_kafka_writer_messages_count`
Number of messages written by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_bytes_count`
Number of bytes written by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_errors_count`
Number of errors occurred during writing

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_write_seconds_sum`
Number of writes performed by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_write_seconds_count`
Total time spent on writes

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_write_seconds_min`
Min time spent on writes

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_write_seconds_avg`
Average time spent on writes

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_write_seconds_max`
Max time spent on writes

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_batch_seconds_count`
Number of batches created by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_batch_seconds_sum`
Total time spent on filling batches

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_seconds_min`
Min time spent on filling batches

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_seconds_avg`
Average time spent on filling batches

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_seconds_max`
Max time spent on filling batches

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_batch_queue_seconds_count`
Number of batches submitted in queue

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_batch_queue_seconds_sum`
Total time spent in queue

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_queue_seconds_min`
Min time spent in queue

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_queue_seconds_avg`
Average time spent in queue

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_queue_seconds_max`
Max time spent in queue

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_batch_size_count`
Number of batches written by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_batch_size_sum`
Total size of written batches

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_size_min`
Min written batch size

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_size_avg`
Average written batch size

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_size_max`
Max written batch size

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_batch_bytes_count`
Number of batches written by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_writer_batch_bytes_sum`
Total bytes of written batches

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_bytes_min`
Min written batch bytes

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_bytes_avg`
Average written batch bytes

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_writer_batch_bytes_max`
Max written batch bytes

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **client_id** - kafka client ID

### Kafka Consumer

> **Warning**
> Due to the specifics of the [library](https://github.com/segmentio/kafka-go) used, producer metrics updates every 15 sec, not on read

> **Warning**
> Due to the specifics of the [library](https://github.com/segmentio/kafka-go/blob/v0.4.43/reader.go#L685s) used, value of 'partition' lablel is always '-1'

#### Counter `plugin_kafka_reader_messages_count`
Number of messages read by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_reader_bytes_count`
Number of bytes read by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_reader_errors_count`
Number of errors occurred during reads

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_reader_rebalances_count`
Number of consumer rebalances

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_reader_timeouts_count`
Number of fetches that ends with timeout

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_reader_fetches_count`
Total number of fetches done by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_offset`
Reader current offset

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_lag`
Reader current lag

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_commit_queue_capacity`
Reader internal commit queue capacity

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_commit_queue_length`
Reader internal commit queue length

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_reader_dial_seconds_count`
Number of dials performed by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_reader_dial_seconds_sum`
Total time spent on dials

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_dial_seconds_min`
Min time spent on dials

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_dial_seconds_avg`
Average time spent on dials

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_dial_seconds_max`
Max time spent on dials

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_reader_read_seconds_count`
Number of reads performed by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_reader_read_seconds_sum`
Total time spent on reads

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_read_seconds_min`
Min time spent on reads

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_read_seconds_avg`
Average time spent on reads

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_read_seconds_max`
Max time spent on reads

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_reader_wait_seconds_count`
Number of message waiting cycles

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_reader_wait_seconds_sum`
Total time spent on waiting for messages

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_wait_seconds_min`
Min time spent on waiting

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_wait_seconds_avg`
Average time spent on waiting

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_wait_seconds_max`
Max time spent on waiting

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_reader_fetch_size_count`
Number of fetches performed by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_reader_fetch_size_sum`
Total messages fetched by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_fetch_size_min`
Min messages fetched by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_fetch_size_avg`
Average messages fetched by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_fetch_size_max`
Max messages fetched by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_reader_fetch_bytes_count`
Number of fetches performed by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Counter `plugin_kafka_reader_fetch_bytes_sum`
Total bytes fetched by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_fetch_bytes_min`
Min bytes fetched by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_fetch_bytes_avg`
Average bytes fetched by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID

#### Gauge `plugin_kafka_reader_fetch_bytes_max`
Max bytes fetched by client

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **topic** - kafka topic name
 - **partition** - partition number
 - **group_id** - consumer group
 - **client_id** - kafka client ID
