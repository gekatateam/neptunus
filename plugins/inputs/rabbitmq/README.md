# RabbitMQ Input Plugin

The `rabbitmq` input plugin reads from RabbitMQ queues and passes each message to configured parser. This plugin requires parser.

Each consumer uses it's own ACK queue into which each consumed message is placed. A message ACKed if all of its events hooks are called or if parser returned zero events.

If ACK queue is full, consuming is suspended until at least one message is ACKed.

## Configuration
```toml
[[inputs]]
  [inputs.rabbitmq]
    # list of RabbitMQ cluster nodes
    # if multiple brokers are specified a random broker will be selected 
    # anytime a connection is established
    brokers = [ "amqp://localhost:5672" ]

    # RabbitMQ vhost to connect
    vhost = "/"

    # https://www.rabbitmq.com/docs/consumers#consumer-tags
    # seven base62-characters long random suffix with dash will be appended on init
    # e.g. `neptunus.rabbitmq.input-95Th48s`
    consumer_tag = "neptunus.rabbitmq.input"

    # https://www.rabbitmq.com/docs/connections#client-provided-names
    connection_name = "neptunus.rabbitmq.input"

    # what plugin shoud do with message if parsing failed
    # - "drop" - immediately ack message without producing any event
    # - "reject" - immediately reject message without producing any event
    # - "consume" - produce one event with message data as event body (it can be accessed using "." path)
    on_parser_error = "reject"

    # authentication credentials for the PLAIN auth
    username = ""
    password = ""

    # if true, incoming message timestamp will be used as event timestamp
    keep_timestamp = false

    # if true, incoming message ID will be used as event ID (if it's not empty)
    keep_message_id = false

    # maximum amount of time a dial will wait for a connect to complete
    dial_timeout = "10s"

    # frequency at which consumer sends the heartbeat update
    heartbeat_interval = "10s"

    # https://www.rabbitmq.com/docs/consumer-prefetch
    prefetch_count = 0

    # maximum length of internal unacked messages queue
    max_undelivered = 10

    # if configured, an event id will be set by data from path
    # expected format - "type:path"
    id_from = "field:path.to.id"

    ## TLS configuration
    # if true, TLS client will be used
    # broker address scheme must be set to `amqps`
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

    # list of of the exchanges to declare
    # if unset, no exchanges will be declared
    [[inputs.rabbitmq.exchanges]]
      # exchange name
      name = "neptunus.rabbitmq.exchange.fanout"

      # exchange type, "direct", "fanout", "topic" or "header"
      type = "fanout"

      # https://www.rabbitmq.com/docs/exchanges#durability
      # https://www.rabbitmq.com/docs/exchanges#auto-deletion
      durable = false
      auto_delete = false

      # if true, passive declaration will be used (exchange must be already exists)
      passive = false

      # optional declaration arguments
      declare_args = { alternate-exchange = "alter-ae" }

    # list of the queues to declare and to consume from
    # at least one queue required
    [[inputs.rabbitmq.queues]]
      # queue name
      # if queue is auto-deleted or exclusive, plugin adds random suffix to configured name
      # this suffix is always seven base62-characters long and appends to name with a dash
      # for example - `neptunus.rabbitmq.events.1-xCKg2kJ`
      name = "neptunus.rabbitmq.events.1"

      # https://www.rabbitmq.com/docs/queues#durability
      # https://www.rabbitmq.com/docs/queues#temporary-queues
      durable = false
      auto_delete = false

      # exclusive queues are only accessible by the connection that declares them 
      # and will be deleted when the connection closes
      exclusive = false

      # if true, rejected message will be queued to be delivered to a consumer 
      # on a different channel
      requeue = false

      # if true, passive declaration will be used (queue must be already exists)
      passive = false

      # optional declaration arguments
      declare_args = { "x-message-ttl" = 60000, "x-max-length" = 10 }

      # optional consumer arguments with specific semantics for the queue or server
      consume_args = { "x-message-ttl" = 60000, "x-max-length" = 10 }

      # optional list of queue bindings
      [[inputs.rabbitmq.queues.bindings]]
        # exchange name to bind
        # must be declared in `exchanges` list
        bind_to = "neptunus.rabbitmq.exchange.fanout"

        # binding routing key
        binding_key = "#"

        # optional binding arguments
        declare_args = { "x-dead-letter-exchange" = "dlq" }

    # a "label name <- header" map
    # if message header exists, it will be saved as configured label
    [inputs.rabbitmq.labelheaders]
      extra-type = "msg-extra-type"

    [inputs.rabbitmq.parser]
        type = "json"
```
