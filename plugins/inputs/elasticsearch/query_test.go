package elasticsearch

import (
	"context"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gekatateam/neptunus/core"
	"strings"
	"testing"
)

func TestQuery_Run(t *testing.T) {
	cfg := elasticsearch.Config{
		Addresses:         []string{"http://localhost:9200"},
		EnableDebugLogger: true,
	}

	var client *elasticsearch.Client
	var err error
	client, err = elasticsearch.NewClient(cfg)
	if err != nil {
		t.Error(fmt.Errorf("failed to create elasticsearch client: %w", err))
		return
	}

	// Verify connection
	info, err := client.Info()
	if err != nil {
		t.Error(fmt.Errorf("failed to connect to elasticsearch: %w", err))
		return
	}
	fmt.Println(info)

	q := &Query{
		BaseInput: &core.BaseInput{},
		Name:      "test",
		Index:     "kibana_sample_data_logs",
		QueryBodyRaw: `{
  "sort": [
    {
      "timestamp": {
        "order": "desc",
        "format": "strict_date_optional_time",
        "unmapped_type": "boolean"
      }
    },
    {
      "_doc": {
        "order": "desc",
        "unmapped_type": "boolean"
      }
    }
  ],
  "track_total_hits": false,
  "fields": [
    {
      "field": "*",
      "include_unmapped": "true"
    },
    {
      "field": "@timestamp",
      "format": "strict_date_optional_time"
    },
    {
      "field": "timestamp",
      "format": "strict_date_optional_time"
    },
    {
      "field": "utc_time",
      "format": "strict_date_optional_time"
    }
  ],
  "size": 500,
  "version": true,
  "script_fields": {},
  "stored_fields": [
    "*"
  ],
  "runtime_mappings": {
    "hour_of_day": {
      "type": "long",
      "script": {
        "source": "emit(doc['timestamp'].value.getHour());"
      }
    }
  },
  "_source": false,
  "query": {
    "bool": {
      "must": [],
      "filter": [
        {
          "range": {
            "timestamp": {
              "format": "strict_date_optional_time",
              "gte": "2025-03-22T17:00:00.000Z",
              "lte": "2025-03-23T16:59:59.999Z"
            }
          }
        }
      ],
      "should": [],
      "must_not": []
    }
  },
  "highlight": {
    "pre_tags": [
      "@kibana-highlighted-field@"
    ],
    "post_tags": [
      "@/kibana-highlighted-field@"
    ],
    "fields": {
      "*": {}
    },
    "fragment_size": 2147483647
  }
}`,
		client: client,
	}

	res, err := client.Search(
		client.Search.WithContext(context.Background()),
		client.Search.WithIndex("kibana_sample_data_logs"),
		client.Search.WithBody(strings.NewReader(q.QueryBodyRaw)),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		t.Error(fmt.Errorf("error response: %s", res.String()))
		return
	}
	defer res.Body.Close()
	fmt.Println(res)

	q.Run()
}
