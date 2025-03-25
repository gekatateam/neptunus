# RabbitMQ Output Plugin
The `rabbitmq` output plugin publish events to RabbitMQ. This plugin requires serializer.

Target exchange takes from an event routing key. Plugin creates one publisher per exchange. **All mesages will be published as non-mandatory and non-immediately!**

Each event will be serialized into a individual message, `batch_*` settings controls **messages** batching.

## Configuration
```toml
[[outputs]]
  [outputs.rabbitmq]
    # list of RabbitMQ cluster nodes
    # if multiple brokers are specified a random broker will be selected 
    # anytime a connection is established
    brokers = [ "amqp://localhost:5672" ]

    # RabbitMQ vhost to connect
    vhost = "/"

    # https://www.rabbitmq.com/docs/connections#client-provided-names
    connection_name = "neptunus.rabbitmq.output"

    # publishing app_id attribute
    application_id = "neptunus.rabbitmq.output"

    # authentication credentials for the PLAIN auth
    username = ""
    password = ""

    # publisher persistence mode; "persistent" or "transient"
    delivery_mode = "persistent"

    # if true, outgoing message timestamp will set from event timestamp
    keep_timestamp = false

    # if true, outgoing message id will set from event id
    keep_message_id = false

    # optional label name, which value will be used as publishing routing key
    routing_label = "routing_key"

    # optional label name, which value will be used as message type
    type_label = "event_type"

    # time after which inactive publishers will be closed
    # if configured value a zero, idle publishers will never be closed
    # if configured value less than 1m but not zero, it will be set to 1m
    idle_timeout = "1h"

    # maximum amount of time a dial will wait for a connect to complete
    dial_timeout = "10s"

    # frequency at which consumer sends the heartbeat update
    heartbeat_interval = "10s"

    # interval between events buffer flushes if buffer length less than it's capacity
    batch_interval = "5s"

    # events buffer size, also, messages batch size
    # if configured value less than 1, it will be set to 1
    batch_buffer = 100

    # maximum number of attempts to send a batch of messages
    # before events will be marked as failed
    # 
    # unACKed messages also cause a retry
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
    # minimum TLS version, not limited by default
    tls_min_version = "TLS12"
    # send the specified TLS server name via SNI
    tls_server_name = "exmple.svc.local"
    # use TLS but skip chain & host verification
    tls_insecure_skip_verify = false

    # a "header -> label" map
    # if event label exists, it will be added as a message header
    [outputs.rabbitmq.headerlabels]
      custom_header = "my_label_name"

    [outputs.rabbitmq.serializer]
      type = "json"
      data_only = true
```
