package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
)

type readersPool struct {
	new func() *topicReader
}

type topicReader struct {
	alias         string
	pipe          string
	topic         string
	labelHeaders  map[string]string

	reader *kafka.Reader
	chSem   chan struct{}
	wgSem   *sync.WaitGroup

	rCtx   context.Context
	out    chan <- *core.Event
	log     *slog.Logger
	parser core.Parser
}

func (r *topicReader) Run() {
FETCH_LOOP:
	for {
		msg, err := r.reader.FetchMessage(r.rCtx)
		now := time.Now()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				r.log.Debug(fmt.Sprintf("consumer for topic %v context canceled", r.topic))
				break FETCH_LOOP
			}

			r.log.Error("fetch error", 
				"error", err,
			)
			metrics.ObserveInputSummary("kafka", r.alias, r.pipe, metrics.EventFailed, time.Since(now))
			continue FETCH_LOOP
		}

		events, err := r.parser.Parse(msg.Value, r.topic)
		if err != nil {
			r.log.Error("parser error", 
				"error", err,
			)
			// IF COMMIT ON PARSER ERROR HERE
			metrics.ObserveInputSummary("kafka", r.alias, r.pipe, metrics.EventFailed, time.Since(now))
			continue FETCH_LOOP
		}

		// here we expect a header value to be a string
		headers := make(map[string]string)
		for _, header := range msg.Headers {
			headers[string(header.Key)] = string(header.Value)
		}

		for _, e := range events {
			r.chSem <- struct{}{}
			r.wgSem.Add(1)

			for label, header := range r.labelHeaders {
				if h, ok := headers[header]; ok {
					e.AddLabel(label, h)
				}
			}

			// HERE ADD TRACKER TO EVENT

			r.out <- e
			r.log.Debug("event accepted",
			slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			metrics.ObserveInputSummary("kafka", r.alias, r.pipe, metrics.EventAccepted, time.Since(now))
			now = time.Now()
		}
	}

	r.log.Info(fmt.Sprintf("consumer for topic %v done, waiting for %v events delivery", r.topic, len(r.chSem)))
	r.wgSem.Wait()

	if err := r.reader.Close(); err != nil {
		r.log.Warn(fmt.Sprintf("consumer for topic %v closed with error", r.topic),
			"error", err,
		)
	} else {
		r.log.Info(fmt.Sprintf("consumer for topic %v closed", r.topic))
	}

	// ENABLE METRICS HANDLE HERE
}
