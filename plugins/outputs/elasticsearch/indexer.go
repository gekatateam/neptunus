package elasticsearch

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/goccy/go-json"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/bulk"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/operationtype"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
)

type indexer struct {
	alias string
	pipe  string

	lastWrite    time.Time
	pipeline     string
	dataOnly     bool
	operation    operationtype.OperationType
	routingLabel string
	timeout      time.Duration

	maxAttempts int
	retryAfter  time.Duration

	client *elasticsearch.TypedClient
	*batcher.Batcher[*core.Event]

	log   *slog.Logger
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

func (i *indexer) LastWrite() time.Time {
	return i.lastWrite
}

func (i *indexer) Push(e *core.Event) {
	i.input <- e
}

func (i *indexer) Run() {
	i.log.Info(fmt.Sprintf("indexer for pipeline %v spawned", i.pipeline))

	i.Batcher.Run(i.input, func(buf []*core.Event) {
		if len(buf) == 0 {
			return
		}
		var (
			now        time.Time         = time.Now()
			sentEvents []measurableEvent = make([]measurableEvent, 0, len(buf))
			req        *bulk.Bulk        = bulk.New(i.client)
		)
		i.lastWrite = now

		if len(i.pipeline) > 0 {
			req = req.Pipeline(i.pipeline)
		}

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
				i.log.Error("event serialization failed, event skipped",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				e.Done()
				metrics.ObserveOutputSummary("elasticsearch", i.alias, i.pipe, metrics.EventFailed, time.Since(now))
				now = time.Now()
				continue
			}

			var routing *string
			if len(i.routingLabel) > 0 {
				if label, ok := e.GetLabel(i.routingLabel); ok {
					routing = &label
				}
			}

			var opErr error
			switch i.operation {
			case operationtype.Create:
				opErr = req.CreateOp(types.CreateOperation{
					Id_:     &e.Id,
					Index_:  &e.RoutingKey,
					Routing: routing,
				}, rawEvent)
			case operationtype.Index:
				opErr = req.IndexOp(types.IndexOperation{
					Id_:     &e.Id,
					Index_:  &e.RoutingKey,
					Routing: routing,
				}, rawEvent)
			}

			if opErr != nil {
				i.log.Error("operation serialization failed, event skipped",
					"error", opErr,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				e.Done()
				metrics.ObserveOutputSummary("elasticsearch", i.alias, i.pipe, metrics.EventFailed, time.Since(now))
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

		res, err := i.perform(req)
		sentEvents[len(sentEvents)-1].spentTime += time.Since(now)
		if err != nil {
			for _, e := range sentEvents {
				i.log.Error("event send failed",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				e.Done()
				metrics.ObserveOutputSummary("elasticsearch", i.alias, i.pipe, metrics.EventFailed, e.spentTime)
			}
			return
		}

		for j, v := range res.Items {
			e := sentEvents[j]
			if errCause := v[i.operation].Error; errCause != nil {
				i.log.Error("event send failed",
					"error", errCause.Type+": "+*errCause.Reason,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				e.Done()
				metrics.ObserveOutputSummary("elasticsearch", i.alias, i.pipe, metrics.EventFailed, e.spentTime)
			} else {
				i.log.Debug("event sent",
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				e.Done()
				metrics.ObserveOutputSummary("elasticsearch", i.alias, i.pipe, metrics.EventAccepted, e.spentTime)
			}
		}
	})

	i.log.Info(fmt.Sprintf("indexer for pipeline %v closed", i.pipeline))
}

func (i *indexer) perform(r *bulk.Bulk) (*bulk.Response, error) {
	var attempts int = 1
	for {
		response, err := i.perform2(r)
		if err == nil {
			return response, nil
		}

		switch {
		case i.maxAttempts > 0 && attempts < i.maxAttempts:
			i.log.Warn(fmt.Sprintf("bulk request attempt %v of %v failed", attempts, i.maxAttempts))
			attempts++
			time.Sleep(i.retryAfter)
		case i.maxAttempts > 0 && attempts >= i.maxAttempts:
			i.log.Error(fmt.Sprintf("bulk request failed after %v attemtps", attempts),
				"error", err,
			)
			return response, err
		default:
			i.log.Error("bulk request failed",
				"error", err,
			)
			time.Sleep(i.retryAfter)
		}
	}
}

func (i *indexer) perform2(r *bulk.Bulk) (*bulk.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), i.timeout)
	defer cancel()
	return r.Do(ctx)
}
