package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
)

type Query struct {
	*core.BaseInput `mapstructure:"-"`
	Name            string                 `mapstructure:"name"`
	Schedule        string                 `mapstructure:"schedule"`
	Index           string                 `mapstructure:"index"`
	QueryBody       map[string]interface{} `mapstructure:"query_body"`
	QueryBodyRaw    string                 `mapstructure:"query_body_raw"`
	TimeoutSeconds  int                    `mapstructure:"timeout_seconds"`

	client *elasticsearch.Client
}

func (q *Query) Run() {
	now := time.Now()
	ctx := context.Background()

	if q.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(q.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	// Convert query body to JSON
	var query []byte
	var err error
	if q.QueryBodyRaw != "" {
		query = []byte(q.QueryBodyRaw)
	} else {
		query, err = json.Marshal(q.QueryBody)
		if err != nil {
			q.Log.Error("failed to marshal query", slog.String("error", err.Error()))
			q.Observe(metrics.EventRejected, time.Since(now))
			return
		}
	}

	// Perform the search request
	req := esapi.SearchRequest{
		Index: []string{q.Index},
		Body:  bytes.NewReader(query),
	}

	res, err := req.Do(ctx, q.client)
	if err != nil {
		q.Log.Error("elasticsearch query failed",
			slog.String("error", err.Error()),
			slog.String("query", q.Name),
		)
		q.Observe(metrics.EventRejected, time.Since(now))
		return
	}
	defer res.Body.Close()

	if res.IsError() {
		q.Log.Error("elasticsearch returned error",
			slog.String("status", res.Status()),
			slog.String("query", q.Name),
		)
		q.Observe(metrics.EventRejected, time.Since(now))
		return
	}

	// Parse the response
	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		q.Log.Error("error parsing response", slog.String("error", err.Error()))
		q.Observe(metrics.EventRejected, time.Since(now))
		return
	}

	// Create event with search results
	e := core.NewEvent(fmt.Sprintf("elasticsearch.%s", q.Name))
	e.SetLabel("index", q.Index)
	e.SetLabel("took_ms", fmt.Sprintf("%v", r["took"]))

	// Add the hits to the event data
	if hits, ok := r["hits"].(map[string]interface{}); ok {
		e.Data = hits
	} else {
		e.Data = r
	}

	q.Out <- e

	q.Log.Debug("elasticsearch event produced",
		slog.Group("event",
			"id", e.Id,
			"key", e.RoutingKey,
			"hits", fmt.Sprintf("%v", getHitCount(r)),
		),
	)
	q.Observe(metrics.EventAccepted, time.Since(now))
}

// Helper function to extract hit count from response
func getHitCount(response map[string]interface{}) int64 {
	if hits, ok := response["hits"].(map[string]interface{}); ok {
		if total, ok := hits["total"].(map[string]interface{}); ok {
			if value, ok := total["value"].(float64); ok {
				return int64(value)
			}
		}
	}
	return 0
}
