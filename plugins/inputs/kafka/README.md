# Kafka Input Plugin

The `kafka` input plugin reads from Kafka and passes each message to configured parser. This plugin requires parser.

Each reader uses its own commit queue into which each fetched message is placed. Every `commit_interval` fetch process paused and queue scanning for uncommitted ready sequence from oldest to newest messages. Largest offset found from the beginning of the queue will be committed.

A message is marked as ready to commit if all of its events hooks are called, if parser returned zero events, or if parsing ended with an error.

If commit queue is full, fetching is suspended until at least one message is committed.

> [!TIP]  
> This plugin may write it's own [metrics](../../../docs/METRICS.md#kafka-consumer)

## Configuration
```toml
[[inputs]]
  [inputs.kafka]
    # if true, plugin client writes it's own metrics
    enable_metrics = false

    # list of kafka cluster nodes
    brokers = [ "localhost:9092" ]

    # unique identifier that the transport communicates to brokers when it sends requests
    client_id = "neptunus.kafka.input"

    # topics to consume
    # topic name will be set as an event routing key
    topics = [ "topic_one", "topic_two" ]

    # if configured, an event id will be set by data from path
    # expected format - "type:path"
    id_from = "field:path.to.id"

    # determines from whence the consumer group should begin consuming
    # when it finds a partition without a committed offset, "first" or "last"
    start_offset = "last"

    # consumer group identifier
    group_id = "neptunus.kafka.input"

    # amount of time the consumer group will be saved by the broker
    group_ttl = "24h"

    # consumer group balancing strategy
    # "range", "round-robin" or "rack-affinity"
    group_balancer = "range"

    # rack where this consumer is running
    # requires for rack-affinity group balancer
    rack = "rack-01"

    # maximum amount of time a dial will wait for a connect to complete
    dial_timeout = "5s"

    # amount of time that may pass without a heartbeat before the coordinator
    # considers the consumer dead and initiates a rebalance
    session_timeout = "30s"

    # amount of time the coordinator will wait for members to join as part of a rebalance
    rebalance_timeout = "30s"

    # frequency at which the reader sends the consumer group heartbeat update
    heartbeat_interval = "3s"

    # amount of time to wait to fetch message from kafka messages batch
    read_batch_timeout = "3s"

    # amount of time to wait for new data to come when fetching batches of messages from kafka
    wait_batch_timeout = "3s"

    # maximum batch size that the consumer will accept
    max_batch_size = "1MiB"

    # maximum length of internal uncommitted messages queue
    max_uncommitted = 100

    # interval between commit queue scans
    commit_interval = "1s"

    # interval between commit retries after error
    commit_retry_interval = "1s"

    ## TLS configuration
    # if true, TLS client will be used
    tls_enable = false
    # trusted root certificates for server
    tls_ca_file = "/etc/neptunus/ca.pem"
    # used for TLS client certificate authentication
    tls_key_file = "/etc/neptunus/key.pem"
    tls_cert_file = "/etc/neptunus/cert.pem"
    # minimum TLS version, not limited by default
    tls_min_version = "TLS12"
    # send the specified TLS server name via SNI
    tls_server_name = "exmple.svc.local"
    # use TLS but skip chain & host verification
    tls_insecure_skip_verify = false

    # SASL settings
    [inputs.kafka.sasl]
      # SASL mechanism, "none", "plain", "scram-sha-256" or "scram-sha-512"
      mechanism = "scram-sha-512"

      # user credentials
      username = ""
      password = ""

    # a "label name -> header" map
    # if message header exists, it will be saved as configured label
    [inputs.kafka.labelheaders]
      length = "Content-Length"
      encoding = "Content-Type"

    [inputs.kafka.parser]
      type = "json"
```
