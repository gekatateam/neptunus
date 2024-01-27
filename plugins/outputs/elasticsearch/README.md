# Elasticsearch Output Plugin

The `elasticsearch` output plugin writes to [Elasticsearch](https://www.elastic.co/guide/en/elasticsearch/reference/current/index.html) via HTTP using [Bulk API](https://www.elastic.co/guide/en/elasticsearch/reference/8.11/docs-bulk.html).

# Configuration
```toml
[[outputs]]
  [outputs.elasticsearch]
    # list of Elasticsearch nodes to use
    # only one will be used on each write
    urls = [ "http://localhost:9200" ]

    # discover nodes periodically
    # with this option it is not necessary to list all nodes in config
    # zero for disable
    discover_interval = "0s"

    # username and password for HTTP Basic Authentication
    username = ""
    password = ""

    # service token for authorization
    # if set, overrides username/password
    service_token = ""

    # base64-encoded token for authorization
    # if set, overrides username/password and service token
    api_key = ""

    # endpoint for the Elastic Service (https://elastic.co/cloud)
    cloud_id = ""

    # SHA256 hex fingerprint given by Elasticsearch on first launch
    cert_fingerprint = ""

    # bulk operation, "create" or "index"
    # data streams supports only "create" operation
    operation = "create"

    # if false, all event will be used as a document
    data_only = true

    # label with ingest pipeline name
    # plugin creates one indexer per unique label value
    pipeline_label = ""

    # label which value will be used to route operations to a specific shard
    routing_label = ""

    # time after which inactive indexers will be closed
    # if configured value a zero, idle producers will never be closed
    # if configured value less than 1m but not zero, it will be set to 1m
    idle_timeout = "1h"

    # interval between events buffer flushes if buffer length less than it's capacity
    batch_interval = "5s"

    # events buffer size
    batch_buffer = 100    

    # gzip compression usage flag
    enable_compression = true

    # timeout for HTTP requests
    request_timeout = "10s"

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
