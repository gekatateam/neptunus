# Elasticsearch Output Plugin

The `elasticsearch` output plugin writes to [Elasticsearch](https://www.elastic.co/guide/en/elasticsearch/reference/current/index.html) via HTTP.

# Configuration
```toml
[[outputs]]
  [outputs.elasticsearch]
    #
    urls = [ "http://localhost:9200" ]

    #
    username = ""
    password = ""

    #
    service_token = ""

    #
    api_key = ""

    #
    cloud_id = ""

    #
    cert_fingerprint = ""

    #
    operation = "create"

    #
    data_only = true

    #
    pipeline_label = ""

    #
    routing_label = ""

    # interval between events buffer flushes if buffer length less than it's capacity
    batch_interval = "5s"

    # events buffer size
    batch_buffer = 100    

    #
    enable_compression = true

    #
    request_interval = "10s"

    #
    idle_timeout = "1h"

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
```
