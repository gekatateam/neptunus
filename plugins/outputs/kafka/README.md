# Kafka Output Plugin
The `kafka` output plugin produces events to Kafka. This plugin requires serializer.

Target topic name takes from an event routing key. Plugin creates one writer per topic.

> [!TIP]  
> This plugin may write it's own [metrics](../../../docs/METRICS.md#kafka-producer)

## Configuration
```toml
[[outputs]]
  [outputs.kafka]
    # if true, plugin client writes it's own metrics
    enable_metrics = false

    # list of kafka cluster nodes
    brokers = [ "localhost:9092" ]

    # unique identifier that the transport communicates to brokers when it sends requests
    client_id = "neptunus.kafka.output"

    # time after which inactive producers will be closed
    # if configured value a zero, idle producers will never be closed
    # if configured value less than 1m but not zero, it will be set to 1m
    idle_timeout = "1h"

    # time limit set for establishing connections to the kafka cluster
    dial_timeout = "5s"

    # timeout for write operations performed by the writer
    write_timeout = "5s"

    # interval between events buffer flushes if buffer length less than it's capacity
    batch_interval = "5s"

    # events buffer size, also, messages batch size
    # if configured value less than 1, it will be set to 1
    batch_buffer = 100

    # kafka message maximum size in bytes
    # too large messages will be dropped
    max_message_size = "1MiB"

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
    # otherwise, it will be automatically set when writing a message
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
    # only messages that ended with:
    # - network timeout error 
    # - retriable error - https://kafka.apache.org/protocol#protocol_error_codes
    # will be used in next attempt
    retry_attempts = 0 # zero for endless attempts

    # interval between retries to (re-)send a batch of messages
    retry_after = "5s"

    ## TLS configuration
    # if true, TLS client will be used
    tls_enable = false
    # trusted root certificates for server
    tls_ca_file = "/etc/neptunus/ca.pem"
    # used for TLS client certificate authentication
    tls_key_file = "/etc/neptunus/key.pem"
    tls_cert_file = "/etc/neptunus/cert.pem"
    # send the specified TLS server name via SNI
    tls_server_name = "exmple.svc.local"
    # use TLS but skip chain & host verification
    tls_insecure_skip_verify = false

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
