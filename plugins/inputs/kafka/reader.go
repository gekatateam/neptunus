package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	orderedmap "github.com/wk8/go-ordered-map/v2"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/ider"
	kafkastats "github.com/gekatateam/neptunus/plugins/common/metrics"
)

type readersPool map[string]*topicReader

type topicReader struct {
	alias    string
	pipe     string
	topic    string
	groupId  string
	clientId string

	enableMetrics bool
	labelHeaders  map[string]string

	reader          *kafka.Reader
	commitSemaphore chan struct{}

	fetchCh  chan kafka.Message
	commitCh chan int64
	exitCh   chan struct{}
	doneCh   chan struct{}

	out    chan<- *core.Event
	log    *slog.Logger
	parser core.Parser
	ider   *ider.Ider
}

func (r *topicReader) Run(rCtx context.Context) {
	if r.enableMetrics {
		kafkastats.RegisterKafkaReader(r.pipe, r.alias, r.topic, r.groupId, r.clientId, func() kafkastats.ReaderStats {
			return kafkastats.ReaderStats{
				ReaderStats:         r.reader.Stats(),
				CommitQueueLenght:   len(r.commitSemaphore),
				CommitQueueCapacity: cap(r.commitSemaphore),
			}
		})
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

		r.log.Debug("message fetched")

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

			r.commitSemaphore <- struct{}{}
			r.fetchCh <- msg
			e.SetHook(func(offset any) {
				r.commitCh <- offset.(int64)
			}, msg.Offset)

			r.ider.Apply(e)
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

	r.log.Info(fmt.Sprintf("consumer for topic %v done, waiting for events delivery", r.topic))
	r.exitCh <- struct{}{}
	<-r.doneCh

	r.log.Info("commit queue is empty now, closing consumer")
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

type trackedMessage struct {
	kafka.Message
	delivered bool
}

type commitController struct {
	commitInterval time.Duration

	reader           *kafka.Reader
	commitQueue      *orderedmap.OrderedMap[int64, *trackedMessage]
	commitSemaphore  chan struct{} // max uncommitted control
	exitIfQueueEmpty bool

	fetchCh  chan kafka.Message // for new messages to push in commitQueue
	commitCh chan int64         // for offsets that ready to be committed
	exitCh   chan struct{}      // for exit preparation signal
	doneCh   chan struct{}      // done signal, channel will be closed just before watch() returns

	log *slog.Logger
}

func (c *commitController) Run() {
	ticker := time.NewTicker(c.commitInterval)

	for {
		select {
		case msg, _ := <-c.fetchCh: // new message fetched
			c.log.Debug(fmt.Sprintf("accepted msg with offset: %v", msg.Offset))

			// add message to queue
			c.commitQueue.Store(msg.Offset, &trackedMessage{
				Message:   msg,
				delivered: false,
			})
		case offset, _ := <-c.commitCh: // an event delivered
			c.log.Debug(fmt.Sprintf("got delivered offset: %v", offset))

			// mark message as delivered
			if m, ok := c.commitQueue.Get(offset); ok {
				m.delivered = true
			} else { // normally it is never happens
				c.log.Error("unexpected case, delivered offset is not in queue; please, report this issue")
			}
		case <-ticker.C: // it is time to commit messages
			c.log.Debug("commit phase started, consuming paused")

			// find the uncommitted sequence from queue beginning
			var offsetsToDelete []int64
			var commitCandidate *kafka.Message
			for pair := c.commitQueue.Oldest(); pair != nil; pair = pair.Next() {
				if !pair.Value.delivered {
					break
				}
				commitCandidate = &pair.Value.Message
				offsetsToDelete = append(offsetsToDelete, pair.Key)
			}

			if commitCandidate != nil {
				c.log.Debug(fmt.Sprintf("got candidate with offset: %v", commitCandidate.Offset))
			BEFORE_COMMIT:
				// TODO: not the best solution because we don't know how library handling consumers rebalancing
				// we need test it
				if err := c.reader.CommitMessages(context.Background(), *commitCandidate); err != nil {
					c.log.Error("offset commit failed",
						"error", err,
					)
					time.Sleep(time.Second) // TODO: make this configurable
					goto BEFORE_COMMIT
				}

				for _, v := range offsetsToDelete {
					c.commitQueue.Delete(v)
					<-c.commitSemaphore
				}

				c.log.Debug(fmt.Sprintf("offset committed: %v, left in queue: %v, left in channel: %v",
					commitCandidate.Offset, c.commitQueue.Len(), len(c.commitSemaphore)))
			}

			if c.exitIfQueueEmpty && c.commitQueue.Len() == 0 {
				ticker.Stop()
				close(c.doneCh)
				return
			}
		case <-c.exitCh:
			c.exitIfQueueEmpty = true
			if c.commitQueue.Len() == 0 {
				ticker.Stop()
				close(c.doneCh)
				return
			}
		}
	}
}
