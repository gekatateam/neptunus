package kafka

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"
	orderedmap "github.com/wk8/go-ordered-map/v2"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/ider"
	kafkastats "github.com/gekatateam/neptunus/plugins/common/metrics"
)

type topicReader struct {
	*core.BaseInput

	topic     string
	partition string
	groupId   string
	clientId  string

	enableMetrics bool
	labelHeaders  map[string]string
	delay         atomic.Int64

	reader          *kafka.Reader
	commitSemaphore chan struct{}

	fetchCh  chan *trackedMessage
	commitCh chan commitMessage
	exitCh   chan struct{}
	doneCh   chan struct{}

	out    chan<- *core.Event
	parser core.Parser
	ider   *ider.Ider
}

func (r *topicReader) Run(rCtx context.Context) {
	if r.enableMetrics {
		kafkastats.RegisterKafkaReader(r.Pipeline, r.Alias, r.topic, r.partition, r.groupId, r.clientId, func() kafkastats.ReaderStats {
			return kafkastats.ReaderStats{
				ReaderStats:         r.reader.Stats(),
				CommitQueueLenght:   len(r.commitSemaphore),
				CommitQueueCapacity: cap(r.commitSemaphore),
				Delay:               r.loadAndResetDelay(),
			}
		})
		defer kafkastats.UnregisterKafkaReader(r.Pipeline, r.Alias, r.topic, r.partition, r.groupId, r.clientId)
	}

	r.Log.Info(fmt.Sprintf("consumer for topic %v and partition %v spawned", r.topic, r.partition))
FETCH_LOOP:
	for {
		msg, err := r.reader.FetchMessage(rCtx)
		now := time.Now()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				r.Log.Debug(fmt.Sprintf("consumer for topic %v and partition %v context canceled", r.topic, r.partition))
				break FETCH_LOOP
			}

			if errors.Is(err, kafka.ErrGenerationEnded) {
				r.Log.Debug(fmt.Sprintf("consumer for topic %v and partition %v generation ended", r.topic, r.partition))
				break FETCH_LOOP
			}

			// https://github.com/segmentio/kafka-go/issues/1286
			if errors.Is(err, io.ErrUnexpectedEOF) {
				continue FETCH_LOOP
			}

			r.Log.Error("fetch error",
				"error", err,
			)
			r.Observe(metrics.EventFailed, time.Since(now))
			continue FETCH_LOOP
		}

		r.delay.Store(time.Since(msg.Time).Milliseconds())
		r.Log.Debug("message fetched",
			"topic", msg.Topic,
			"offset", strconv.FormatInt(msg.Offset, 10),
			"partition", strconv.Itoa(msg.Partition),
		)

		events, err := r.parser.Parse(msg.Value, r.topic)
		if err != nil {
			r.Log.Error("parser error, message marked as ready to be commited",
				"error", err,
				"topic", msg.Topic,
				"offset", strconv.FormatInt(msg.Offset, 10),
				"partition", strconv.Itoa(msg.Partition),
			)

			r.commitSemaphore <- struct{}{}
			r.fetchCh <- &trackedMessage{
				Message:   msg,
				events:    0,
				delivered: true, // if parser fails, commit message
			}

			r.Observe(metrics.EventFailed, time.Since(now))
			continue FETCH_LOOP
		}

		// here we expect a header value to be a string
		headers := make(map[string]string)
		for _, header := range msg.Headers {
			headers[string(header.Key)] = string(header.Value)
		}

		r.commitSemaphore <- struct{}{}
		r.fetchCh <- &trackedMessage{
			Message:   msg,
			events:    len(events),
			delivered: len(events) == 0, // if parser returns zero events, commit message
		}

		for _, e := range events {
			e.SetLabel("offset", strconv.FormatInt(msg.Offset, 10))
			e.SetLabel("partition", strconv.Itoa(msg.Partition))
			for label, header := range r.labelHeaders {
				if h, ok := headers[header]; ok {
					e.SetLabel(label, h)
				}
			}

			e.AddHook(func() {
				r.commitCh <- commitMessage{
					partition: msg.Partition,
					offset:    msg.Offset,
				}
			})

			r.ider.Apply(e)
			r.out <- e
			r.Log.Debug("event accepted",
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			r.Observe(metrics.EventAccepted, time.Since(now))
			now = time.Now()
		}
	}

	r.Log.Info(fmt.Sprintf("consumer for topic %v and partition %v done, waiting for events delivery", r.topic, r.partition))
	select {
	// multiple goroutines can write to this channel, but only one reads
	// this is a bit dirty hack that doesn't block forever second and other writing goroutines
	case r.exitCh <- struct{}{}:
	default:
	}

	<-r.doneCh

	r.Log.Info(fmt.Sprintf("commit queue is empty now, closing consumer for topic %v and partition %v", r.topic, r.partition))
	if err := r.reader.Close(); err != nil {
		r.Log.Warn(fmt.Sprintf("consumer for topic %v and partition %v closed with error", r.topic, r.partition),
			"error", err,
		)
	} else {
		r.Log.Info(fmt.Sprintf("consumer for topic %v and partition %v closed", r.topic, r.partition))
	}
}

func (r *topicReader) loadAndResetDelay() int64 {
	defer r.delay.Store(0)
	return r.delay.Load()
}

type trackedMessage struct {
	kafka.Message
	events    int
	delivered bool
}

type commitMessage struct {
	partition int
	offset    int64
}

type commitController struct {
	gen              *kafka.Generation
	commitQueues     map[int]*orderedmap.OrderedMap[int64, *trackedMessage]
	commitSemaphore  chan struct{} // max uncommitted control
	exitIfQueueEmpty bool
	commitInterval   time.Duration

	fetchCh  chan *trackedMessage // for new messages to push in commitQueue
	commitCh chan commitMessage   // for offsets that ready to be committed
	exitCh   chan struct{}        // for exit preparation signal
	doneCh   chan struct{}        // done signal, channel will be closed just before watch() returns

	log *slog.Logger
}

func (c *commitController) Run() {
	ticker := time.NewTicker(c.commitInterval)

	for {
		select {
		case msg := <-c.fetchCh: // new message fetched
			c.log.Debug(fmt.Sprintf("accepted msg with partition: %v, offset: %v", msg.Partition, msg.Offset))

			q, ok := c.commitQueues[msg.Partition]
			if !ok {
				q = orderedmap.New[int64, *trackedMessage](
					orderedmap.WithCapacity[int64, *trackedMessage](cap(c.commitSemaphore)),
				)
				c.commitQueues[msg.Partition] = q
				c.log.Info(fmt.Sprintf("local queue created for partition: %v", msg.Partition))
			}

			// add message to queue
			if _, ok := q.Get(msg.Offset); ok { // normally it is never happens
				c.log.Warn(fmt.Sprintf("duplicate messsage detected with partition: %v, offset: %v; "+
					"there may have been after consumers rebalancing; please, report this issue",
					msg.Partition, msg.Offset))
				//lint:ignore S1005 explicitly indicates reading from the channel, not waiting
				_ = <-c.commitSemaphore
			} else {
				q.Store(msg.Offset, msg)
			}
		case msg := <-c.commitCh: // an event delivered
			c.log.Debug(fmt.Sprintf("got delivered partition: %v, offset: %v", msg.partition, msg.offset))

			// mark message as delivered
			if m, ok := c.commitQueues[msg.partition].Get(msg.offset); ok {
				if m.events == 0 {
					m.delivered = true
				} else {
					m.events--
					if m.events == 0 {
						m.delivered = true
					}
				}
			} else { // normally it is never happens too
				c.log.Error("unexpected case, delivered offset is not in queue; please, report this issue")
			}
		case <-ticker.C: // it is time to commit messages
			c.log.Info("commit phase started, consuming paused")

			for _, q := range c.commitQueues {
				// find the uncommitted sequence from queue beginning
				var offsetsToDelete []int64
				var commitCandidate *kafka.Message
				for pair := q.Oldest(); pair != nil; pair = pair.Next() {
					if !pair.Value.delivered {
						break
					}
					commitCandidate = &pair.Value.Message
					offsetsToDelete = append(offsetsToDelete, pair.Key)
				}

				if commitCandidate != nil {
					c.log.Debug(fmt.Sprintf("got candidate with partition: %v, offset: %v",
						commitCandidate.Partition, commitCandidate.Offset))
					// BEFORE_COMMIT:
					err := c.gen.CommitOffsets(map[string]map[int]int64{
						commitCandidate.Topic: {
							commitCandidate.Partition: commitCandidate.Offset + 1,
						},
					})
					if err != nil {
						c.log.Error("offset commit failed; cleaning queues from delivered events",
							"error", err,
						)
					}

					for _, v := range offsetsToDelete {
						q.Delete(v)
						//lint:ignore S1005 explicitly indicates reading from the channel, not waiting
						_ = <-c.commitSemaphore
					}

					c.log.Info(fmt.Sprintf("committed/cleaned partition: %v, offset: %v; "+
						"left in local queue: %v, left in global queue: %v",
						commitCandidate.Partition, commitCandidate.Offset, q.Len(), len(c.commitSemaphore)))
				}
			}

			c.log.Info("commit phase completed, consuming unblocked")

			if c.exitIfQueueEmpty && len(c.commitSemaphore) == 0 {
				ticker.Stop()
				close(c.doneCh)
				return
			}
		case <-c.exitCh:
			c.log.Info(fmt.Sprintf("left in global queue: %v", len(c.commitSemaphore)))
			c.exitIfQueueEmpty = true
			if len(c.commitSemaphore) == 0 {
				ticker.Stop()
				close(c.doneCh)
				return
			}
		}
	}
}
