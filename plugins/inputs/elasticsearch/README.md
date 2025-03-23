# Elasticsearch Input Plugin

This plugin queries Elasticsearch on a schedule and emits events containing the search results.

## Configuration

```yaml
inputs:
  - type: elasticsearch
    location: "UTC"
    address:
      - "http://localhost:9200"
    username: "elastic"  # Optional
    password: "changeme" # Optional
    api_key: ""          # Optional, alternative to username/password
    cloud_id: ""         # Optional, for Elastic Cloud deployments
    queries:
      - name: "errors"
        schedule: "*/5 * * * * *"  # Run every 5 seconds
        index: "logs-*"
        timeout_seconds: 30
        query_body:
          query:
            bool:
              must:
                - match:
                    level: "error"
              filter:
                - range:
                    "@timestamp":
                      gte: "now-5m"
      - name: "high_cpu"
        schedule: "0 * * * * *"    # Run every minute
        index: "metrics-*"
        query_body:
          query:
            range:
              system.cpu.total.pct:
                gt: 0.8
```

## Parameters

- `location`: Timezone location (default: "UTC")
- `address`: List of Elasticsearch node URLs
- `username`: Elasticsearch username (if using basic auth)
- `password`: Elasticsearch password (if using basic auth)
- `api_key`: Elasticsearch API key (alternative to username/password)
- `cloud_id`: Elastic Cloud ID (for cloud deployments)

### Query Parameters

- `name`: Unique name for the query
- `schedule`: Cron expression (including seconds)
- `index`: Elasticsearch index pattern to query
- `query_body`: Elasticsearch query in JSON format
- `timeout_seconds`: Query timeout in seconds (default: 0, no timeout)

## Output Events

Events are emitted with the routing key format `elasticsearch.{query_name}`. Each event contains:

- Labels:
  - `index`: The index pattern queried
  - `took_ms`: Time in milliseconds Elasticsearch took to execute the query

- Data: Contains the "hits" section from the Elasticsearch response, including:
  - `total`: Object with count and relation of matching documents
  - `max_score`: Maximum relevance score
  - `hits`: Array of matching documents
