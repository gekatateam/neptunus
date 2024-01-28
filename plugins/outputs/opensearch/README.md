# Opensearch Output Plugin

The `opensearch` output plugin writes to [Opensearch](https://opensearch.org/docs/latest/) via HTTP using [Bulk API](https://opensearch.org/docs/2.11/api-reference/document-apis/bulk/).

# Configuration
```toml
[[outputs]]
  [outputs.opensearch]
    # list of Opensearch nodes to use
    # only one will be used on each write
    urls = [ "http://localhost:9200" ]

    # discover nodes periodically
    # with this option it is not necessary to list all nodes in config
    # zero for disable
    discover_interval = "0s"

    # username and password for HTTP Basic Authentication
    username = ""
    password = ""

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

    # maximum number of attempts to execute bulk request
    # before events will be marked as failed
    # 
    # only requests that ended with:
    # - network timeout error 
    # - HTTP 4xx, 5xx codes
    # will be retried
    retry_attempts = 0 # zero for endless attempts

    # interval between retries to execute bulk request
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
```
