# Kafka Output Plugin
The `kafka` output plugin produces events to Kafka. This plugin requires serializer.

Target topic name takes from an event routing key.

> **Note**
> This plugin may write it's own [metrics](../../../docs/METRICS.md#TODO_KAFKA_PRODUCER_METRICS)

## Configuration
```toml
[[outputs]]
  [outputs.kafka]
    # if true, plugin client writes it's own metrics
    enable_metrics = false

    # list of kafka cluster nodes
    brokers = [ "localhost:9092" ]

    # unique identifier that the transport communicates to the brokers 
    # when it sends requests
    # value must be unique across the app
    client_id = "neptunus.kafka.output.{{ PLUGIN_ID }}"

    # time limit set for establishing connections to the kafka cluster
    dial_timeout = "5s"

    # timeout for write operations performed by the writer
    write_timeout = "5s"

    # time limit on how often incomplete message batches will be flushed from kafka client to broker
    batch_timeout = "100ms"

    # interval between events buffer flushes
    interval = "5s"

    # events buffer size
    buffer = 100

    # kafka message maximum size in bytes
    # too large messages will be dropped
    max_message_size = 1_048_576 # 1 MiB

    # when true, client create topics if they are not exists
    topics_autocreate = false

    # compression codec to be used to compress messages
    # "none", "gzip", "snappy", "lz4", "zstd"
    # if "none", messages are not compressed
    compression = "none"

    # ack mode
    # "none" - fire-and-forget, do not wait for acknowledgements
    # "one" - wait for the leader to acknowledge the writes
    # "all" - wait for the full ISR to acknowledge the writes
    required_acks = "one"

    # if true, message timestamp takes from event timestamp
    # otherwise, it will be automatically set when writing the message
    keep_timestamp = false

    # messages partition balancer
    # "round-robin" - https://pkg.go.dev/github.com/segmentio/kafka-go@v0.4.43#RoundRobin
    # "least-bytes" - https://pkg.go.dev/github.com/segmentio/kafka-go@v0.4.43#LeastBytes
    # "fnv-1a-reference" - https://pkg.go.dev/github.com/segmentio/kafka-go@v0.4.43#ReferenceHash
    # "fnv-1a" - https://pkg.go.dev/github.com/segmentio/kafka-go@v0.4.43#Hash
    # "consistent", "consistent-random" - https://pkg.go.dev/github.com/segmentio/kafka-go@v0.4.43#CRC32Balancer
    # "murmur2", "murmur2-random" - https://pkg.go.dev/github.com/segmentio/kafka-go@v0.4.43#Murmur2Balancer
    # or "label" - takes partition number from configured label
    partition_balancer = "least-bytes"

    # name of label for "label" balancer
    # if label not exists, not a number or not in topic partitions list
    # 0 partition will be used
    partition_label = ""

    # name of label used as message key
    key_label = ""

    # maximum number of attempts to send a batch of messages
    # before event will be marked as failed
    # 
    # only messages that ended with a retriable error will be used in next attempt
    # https://kafka.apache.org/protocol#protocol_error_codes
    max_attempts = 0 # zero for endless attempts

    # interval between retries to (re-)send a batch of messages
    retry_after = "5s"

    # SASL settings
    [outputs.kafka.sasl]
      # SASL mechanism, "none", "plain", "scram-sha-256" or "scram-sha-512"
      mechanism = "scram-sha-512"

      # user credentials
      username = ""
      password = ""

    # a "header -> label" map
    # if event label exists, it will be added as a message header
    [outputs.kafka.headerlabels]
      custom_header = "my_label_name"

    [outputs.kafka.serializer]
      type = "json"
      data_only = true
```
