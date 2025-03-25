package opensearch

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/goccy/go-json"
	"github.com/opensearch-project/opensearch-go/v3/opensearchapi"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	"github.com/gekatateam/neptunus/plugins/common/esopensearch"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
)

const defaultBufferSize = 4096

type indexer struct {
	*core.BaseOutput

	pipeline     string
	dataOnly     bool
	operation    string
	routingLabel string
	timeout      time.Duration

	client *opensearchapi.Client
	*batcher.Batcher[*core.Event]
	*retryer.Retryer

	input chan *core.Event
}

type measurableEvent struct {
	*core.Event
	spentTime time.Duration
}

func (i *indexer) Close() error {
	close(i.input)
	return nil
}

func (i *indexer) Push(e *core.Event) {
	i.input <- e
}

func (i *indexer) Run() {
	i.Log.Info(fmt.Sprintf("indexer for pipeline %v spawned", i.pipeline))

	i.Batcher.Run(i.input, func(buf []*core.Event) {
		if len(buf) == 0 {
			return
		}
		var (
			now        time.Time              = time.Now()
			sentEvents []measurableEvent      = make([]measurableEvent, 0, len(buf))
			body       *esopensearch.BulkBody = &esopensearch.BulkBody{Buffer: bytes.NewBuffer(make([]byte, 0, defaultBufferSize))}
			req        *opensearchapi.BulkReq = &opensearchapi.BulkReq{Params: opensearchapi.BulkParams{Pipeline: i.pipeline}}
		)

		for _, e := range buf {
			var (
				rawEvent []byte
				err      error
			)

			if i.dataOnly {
				rawEvent, err = json.Marshal(e.Data)
			} else {
				rawEvent, err = json.Marshal(e)
			}

			if err != nil {
				i.Log.Error("event serialization failed, event skipped",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				i.Done <- e
				i.Observe(metrics.EventFailed, time.Since(now))
				now = time.Now()
				continue
			}

			var routing string
			if len(i.routingLabel) > 0 {
				if label, ok := e.GetLabel(i.routingLabel); ok {
					routing = label
				}
			}

			var opErr error
			switch i.operation {
			case "create":
				opErr = body.CreateOp(rawEvent, esopensearch.BulkOp{
					Index:   e.RoutingKey,
					Id:      e.Id,
					Routing: routing,
				})
			case "index":
				opErr = body.IndexOp(rawEvent, esopensearch.BulkOp{
					Index:   e.RoutingKey,
					Id:      e.Id,
					Routing: routing,
				})
			}

			if opErr != nil {
				i.Log.Error("operation serialization failed, event skipped",
					"error", opErr,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				i.Done <- e
				i.Observe(metrics.EventFailed, time.Since(now))
				now = time.Now()
				continue
			}

			sentEvents = append(sentEvents, measurableEvent{
				Event:     e,
				spentTime: time.Since(now),
			})
			now = time.Now()
		}

		// bulk request requires body
		// if sentEvents is empty, than all events preparation failed
		if len(sentEvents) == 0 {
			return
		}

		res, err := i.perform(req, body)
		sentEvents[len(sentEvents)-1].spentTime += time.Since(now)
		if err != nil {
			for _, e := range sentEvents {
				i.Log.Error("event send failed",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				i.Done <- e.Event
				i.Observe(metrics.EventFailed, e.spentTime)
			}
			return
		}

		for j, v := range res.Items {
			e := sentEvents[j]
			i.Done <- e.Event
			if errCause := v[i.operation].Error; errCause != nil {
				i.Log.Error("event send failed",
					"error", errCause.Type+": "+errCause.Reason,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				i.Observe(metrics.EventFailed, e.spentTime)
			} else {
				i.Log.Debug("event sent",
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				i.Observe(metrics.EventAccepted, e.spentTime)
			}
		}
	})

	i.Log.Info(fmt.Sprintf("indexer for pipeline %v closed", i.pipeline))
}

func (i *indexer) perform(r *opensearchapi.BulkReq, b *esopensearch.BulkBody) (*opensearchapi.BulkResp, error) {
	var staticBody = b.Bytes() // cache body for retries
	var bulkResponse *opensearchapi.BulkResp

	return bulkResponse, i.Retryer.Do("bulk request", i.Log, func() error {
		r.Body = bytes.NewReader(bytes.Clone(staticBody))

		ctx, cancel := context.WithTimeout(context.Background(), i.timeout)
		response, err := i.client.Bulk(ctx, *r)
		if err == nil { // http error already checked by client here - https://github.com/opensearch-project/opensearch-go/blob/v3.0.0/opensearchapi/opensearchapi.go#L110
			cancel()
			bulkResponse = response
			return nil
		}

		cancel()
		return err
	})
}
