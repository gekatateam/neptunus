package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
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

	topic    string
	groupId  string
	clientId string

	enableMetrics bool
	labelHeaders  map[string]string

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
		kafkastats.RegisterKafkaReader(r.Pipeline, r.Alias, r.topic, r.groupId, r.clientId, func() kafkastats.ReaderStats {
			return kafkastats.ReaderStats{
				ReaderStats:         r.reader.Stats(),
				CommitQueueLenght:   len(r.commitSemaphore),
				CommitQueueCapacity: cap(r.commitSemaphore),
			}
		})
	}

	r.Log.Info(fmt.Sprintf("consumer for topic %v spawned", r.topic))

FETCH_LOOP:
	for {
		msg, err := r.reader.FetchMessage(rCtx)
		now := time.Now()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				r.Log.Debug(fmt.Sprintf("consumer for topic %v context canceled", r.topic))
				break FETCH_LOOP
			}

			r.Log.Error("fetch error",
				"error", err,
			)
			r.Observe(metrics.EventFailed, time.Since(now))
			continue FETCH_LOOP
		}

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

	r.Log.Info(fmt.Sprintf("consumer for topic %v done, waiting for events delivery", r.topic))
	r.exitCh <- struct{}{}
	<-r.doneCh

	r.Log.Info("commit queue is empty now, closing consumer")
	if err := r.reader.Close(); err != nil {
		r.Log.Warn(fmt.Sprintf("consumer for topic %v closed with error", r.topic),
			"error", err,
		)
	} else {
		r.Log.Info(fmt.Sprintf("consumer for topic %v closed", r.topic))
	}

	if r.enableMetrics {
		kafkastats.UnregisterKafkaReader(r.Pipeline, r.Alias, r.topic, r.groupId, r.clientId)
	}
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
	commitInterval time.Duration

	reader           *kafka.Reader
	commitQueues     map[int]*orderedmap.OrderedMap[int64, *trackedMessage]
	commitSemaphore  chan struct{} // max uncommitted control
	exitIfQueueEmpty bool

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
				c.log.Info(fmt.Sprintf("ordered queue created for partition: %v", msg.Partition))
			}

			// add message to queue
			if _, ok := q.Get(msg.Offset); ok {
				c.log.Warn(fmt.Sprintf("duplicate messsage detected with partition: %v, offset: %v;"+
					" there may have been after consumers rebalancing", msg.Partition, msg.Offset))
				//lint:ignore S1005 explicitly indicates reading from the channel, not waiting
				_ = <-c.commitSemaphore
			} else { // normally it is never happens
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
			} else { // normally it is never happens
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
					c.log.Debug(fmt.Sprintf("got candidate with partition: %v, offset: %v", commitCandidate.Partition, commitCandidate.Offset))
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
						q.Delete(v)
						//lint:ignore S1005 explicitly indicates reading from the channel, not waiting
						_ = <-c.commitSemaphore
					}

					c.log.Info(fmt.Sprintf("committed partition: %v, offset: %v; left in local queue: %v, left in global queue: %v",
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
