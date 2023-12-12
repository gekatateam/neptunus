package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	orderedmap "github.com/wk8/go-ordered-map/v2"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/limitedwaitgroup"
	kafkastats "github.com/gekatateam/neptunus/plugins/common/metrics"
)

type trackedMessage struct {
	kafka.Message
	delivered bool
}

type readersPool map[string]*topicReader

type topicReader struct {
	alias    string
	pipe     string
	topic    string
	groupId  string
	clientId string

	enableMetrics bool
	labelHeaders  map[string]string

	reader *kafka.Reader
	sem    *limitedwaitgroup.LimitedWaitGroup
	cQueue *orderedmap.OrderedMap[int64, *trackedMessage]
	cMutex *sync.Mutex

	out    chan<- *core.Event
	log    *slog.Logger
	parser core.Parser
}

func (r *topicReader) Run(rCtx context.Context) {
	if r.enableMetrics {
		kafkastats.RegisterKafkaReader(r.pipe, r.alias, r.topic, r.groupId, r.clientId, r.reader.Stats)
	}

	r.log.Info(fmt.Sprintf("consumer for topic %v spawned", r.topic))

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

			r.sem.Add() // increase semaphore
			r.cMutex.Lock()
			r.cQueue.Store(msg.Offset, &trackedMessage{
				Message:   msg,
				delivered: false,
			}) // add message to queue
			r.cMutex.Unlock()

			e.SetHook(func(offset any) { // set hook to tracker
				r.cMutex.Lock() 
				if m, ok := r.cQueue.Get(offset.(int64)); ok {
					m.delivered = true
				}
				// всё, что дальше, следует выполнять в отдельной горутине по шедулеру
				var commitCandidate *kafka.Message
				for pair := r.cQueue.Oldest(); pair != nil; pair = pair.Next() {
					if !pair.Value.delivered {
						break
					}
					commitCandidate = &pair.Value.Message
				}

				if commitCandidate != nil {
				BEFORE_COMMIT: // есть много вопросов к обработке ошибок, в т.ч. при перебалансировке консьюмеров
					if err := r.reader.CommitMessages(context.Background(), *commitCandidate); err != nil {
						r.log.Error("offset commit failed, stream locked until successfull commit", 
							"error", err,
						)
						time.Sleep(time.Second)
						goto BEFORE_COMMIT
					}

					for pair := r.cQueue.Oldest(); pair != nil; pair = pair.Next() {
						if pair.Key <= commitCandidate.Offset {
							r.cQueue.Delete(pair.Key)
						}
					}
				}
				
				r.cMutex.Unlock()
			}, msg.Offset)

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

type commitController struct {
	cQueue *orderedmap.OrderedMap[int64, *trackedMessage]
	reader *kafka.Reader

	fetchCh chan kafka.Message
	doneCh  chan int64

	log *slog.Logger
}

func (c *commitController) watch() {
WATCH_LOOP:
	for {
		select {
		case msg, ok := <- c.fetchCh:
			if !ok {
				c.fetchCh = nil
				continue WATCH_LOOP
			}

			// add message to queue
			c.cQueue.Store(msg.Offset, &trackedMessage{
				Message:   msg,
				delivered: false,
			}) 
		case offset, ok := <- c.doneCh:
			if !ok {
				c.doneCh = nil
				continue WATCH_LOOP
			}

			// mark message as delivered
			if m, ok := c.cQueue.Get(offset); ok {
				m.delivered = true
			}

			// find the longest uncommitted sequence
			var commitCandidate *kafka.Message
			for pair := c.cQueue.Oldest(); pair != nil; pair = pair.Next() {
				if !pair.Value.delivered {
					break
				}
				commitCandidate = &pair.Value.Message
			}

			if commitCandidate != nil {
			BEFORE_COMMIT:
				if err := c.reader.CommitMessages(context.Background(), *commitCandidate); err != nil {
					c.log.Error("offset commit failed", 
						"error", err,
					)
					time.Sleep(time.Second)
					goto BEFORE_COMMIT
				}

				for pair := c.cQueue.Oldest(); pair != nil; pair = pair.Next() {
					if pair.Key <= commitCandidate.Offset {
						c.cQueue.Delete(pair.Key)
					}
				}
			}
		default:
			// 
		}
	}
}
