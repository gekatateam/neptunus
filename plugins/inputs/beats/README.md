# Beats Input Plugin

The `beats` input plugin enables Neptunus to receive events from the [Elastic Beats](https://www.elastic.co/beats). Configure beat Logstash output to send data to this plugin.

Plugin produces events with `beats.{{ [@metadata][beat] }}` routing key, e.g. `beats.heartbeat` or `beats.filebeat`.

## Configuration
```toml
[[inputs]]
  [inputs.beats]
    # address and port to host Lumberjack listener on
    address = ":5044"

    # keepalive interval returning an ACK of length 0 to lumberjack client, 
    # notifying clients the batch being still active
    keepalive_timeout = "3s"

    # server read and write operations timeout
    network_timeout = "30s"

    # number of workers to be used to process incoming Beats requests
    # default value is number of CPU cores
    num_workers = 8

    # if false, recevied batch acknowledges after reading
    # otherwise, worker will wait for events to be delivered
    ack_on_delivery = true

    # if true, incoming event "@timestamp" field will be used as event timestamp
    keep_timestamp = true

    # if configured, an event id will be set by data from path
    # expected format - "type:path"
    id_from = "field:path.to.id"

    ## TLS configuration
    # if true, TLS listener will be used
    tls_enable = false
    # service key and certificate
    tls_key_file = "/etc/neptunus/key.pem"
    tls_cert_file = "/etc/neptunus/cert.pem"
    # one or more allowed client CA certificate file names to
    # enable mutually authenticated TLS connections
    tls_allowed_cacerts = [ "/etc/neptunus/clientca.pem" ]
    # minimum and maximum TLS version accepted by the service
    # not limited by default
    tls_min_version = "TLS12"
    tls_max_version = "TLS13"

    # a "label name <- metadata key" map
    # if metadata key exists, it will be saved as configured label
    [inputs.beats.labelheaders]
      beat_version = "version"
      event_type = "type"
```
