package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/wk8/go-ordered-map/v2"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	kafkastats "github.com/gekatateam/neptunus/plugins/common/metrics"
)

type topicReader struct {
	alias         string
	pipe          string
	topic         string
	groupId string
	clientId string

	enableMetrics bool
	labelHeaders  map[string]string

	reader *kafka.Reader
	sem    *commitSemaphore
	cQueue *orderedmap.OrderedMap[int64, kafka.Message]

	out    chan <- *core.Event
	log     *slog.Logger
	parser core.Parser
}

func (r *topicReader) Run(rCtx context.Context) {
	if r.enableMetrics {
		kafkastats.RegisterKafkaReader(r.pipe, r.alias, r.topic, r.groupId, r.clientId, r.reader.Stats)
	}

FETCH_LOOP:
	for {
		msg, err := r.reader.FetchMessage(rCtx)
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
			
			metrics.ObserveInputSummary("kafka", r.alias, r.pipe, metrics.EventFailed, time.Since(now))
			continue FETCH_LOOP
		}

		// here we expect a header value to be a string
		headers := make(map[string]string)
		for _, header := range msg.Headers {
			headers[string(header.Key)] = string(header.Value)
		}

		for _, e := range events {
			for label, header := range r.labelHeaders {
				if h, ok := headers[header]; ok {
					e.AddLabel(label, h)
				}
			}

			// HERE ADD TRACKER TO EVENT

			r.sem.Add() // 
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

	r.log.Info(fmt.Sprintf("consumer for topic %v done, waiting for %v events delivery", r.topic, r.sem.Len()))
	r.sem.Wait()

	if err := r.reader.Close(); err != nil {
		r.log.Warn(fmt.Sprintf("consumer for topic %v closed with error", r.topic),
			"error", err,
		)
	} else {
		r.log.Info(fmt.Sprintf("consumer for topic %v closed", r.topic))
	}

	if r.enableMetrics {
		kafkastats.UnregisterKafkaReader(r.pipe, r.alias, r.topic, r.groupId, r.clientId)
	}
}

type commitSemaphore struct {
	ch chan struct{}
	wg *sync.WaitGroup
}

func (s *commitSemaphore) Add() {
	s.ch <- struct{}{}
	s.wg.Add(1)
}

func (s *commitSemaphore) Done() {
	<- s.ch
	s.wg.Done()
}

func (s *commitSemaphore) Len() int {
	return len(s.ch)
}

func (s *commitSemaphore) Cap() int {
	return cap(s.ch)
}

func (s *commitSemaphore) Wait() {
	s.wg.Wait()
}
