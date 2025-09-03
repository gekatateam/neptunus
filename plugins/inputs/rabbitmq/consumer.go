package rabbitmq

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/convert"
	"github.com/gekatateam/neptunus/plugins/common/elog"
	"github.com/gekatateam/neptunus/plugins/common/ider"
)

type trackedDelivery struct {
	amqp.Delivery
	events int
}

type consumer struct {
	*core.BaseInput

	keepTimestamp bool
	keepMessageId bool
	onParserError string
	labelHeaders  map[string]string

	ider         *ider.Ider
	parser       core.Parser
	queue        Queue
	input        <-chan amqp.Delivery
	ackSemaphore chan struct{}

	fetchCh chan trackedDelivery
	ackCh   chan uint64
	exitCh  chan struct{}
	doneCh  chan struct{}
}

func (c *consumer) Run() {
	c.Log.Info(fmt.Sprintf("consumer for quene %v started", c.queue.Name))

CONSUME_LOOP:
	for msg := range c.input {
		now := time.Now()

		c.Log.Debug("message consumed",
			msgLogAttrs(msg, "queue", c.queue.Name)...,
		)

		events, err := c.parser.Parse(msg.Body, c.queue.Name)
		if err != nil {
			c.Log.Error("parsing failed",
				msgLogAttrs(msg, "error", err, "queue", c.queue.Name)...,
			)

			if c.onParserError == "reject" {
				msg.Reject(c.queue.Requeue)
				c.Observe(metrics.EventFailed, time.Since(now))
				continue CONSUME_LOOP
			}

			if c.onParserError == "drop" {
				c.ackSemaphore <- struct{}{}
				c.fetchCh <- trackedDelivery{
					Delivery: msg,
					events:   0,
				}
				c.ackCh <- msg.DeliveryTag

				c.Observe(metrics.EventFailed, time.Since(now))
				continue CONSUME_LOOP
			}

			if c.onParserError == "consume" {
				e := core.NewEventWithData(c.queue.Name, msg.Body)
				e.StackError(err)
				events = []*core.Event{e}
			}
		}

		if len(events) == 0 {
			c.Log.Debug("parser returns zero events, message will be acked",
				msgLogAttrs(msg, "queue", c.queue.Name)...,
			)

			c.ackSemaphore <- struct{}{}
			c.fetchCh <- trackedDelivery{
				Delivery: msg,
				events:   0,
			}
			c.ackCh <- msg.DeliveryTag

			c.Observe(metrics.EventAccepted, time.Since(now))
			continue CONSUME_LOOP
		}

		c.ackSemaphore <- struct{}{}
		c.fetchCh <- trackedDelivery{
			Delivery: msg,
			events:   len(events),
		}

		for _, e := range events {
			e.SetLabel("deliverytag", strconv.FormatUint(msg.DeliveryTag, 10))
			e.SetLabel("consumertag", msg.ConsumerTag)
			e.SetLabel("exchange", msg.Exchange)
			e.SetLabel("routingkey", msg.RoutingKey)
			e.SetLabel("redelivered", strconv.FormatBool(msg.Redelivered))
			for label, header := range c.labelHeaders {
				if h, ok := msg.Headers[header]; ok {
					val, err := convert.AnyToString(h)
					if err != nil {
						c.Log.Warn(fmt.Sprintf("cannot convert header %v:%v to string", header, h),
							msgLogAttrs(msg, "error", err, "queue", c.queue.Name)...,
						)
						continue
					}
					e.SetLabel(label, val)
				}
			}

			if c.keepTimestamp {
				e.Timestamp = msg.Timestamp
			}

			if c.keepMessageId && len(msg.MessageId) > 0 {
				e.Id = msg.MessageId
			}

			e.AddHook(func() {
				c.ackCh <- msg.DeliveryTag
			})

			c.ider.Apply(e)
			c.Out <- e
			c.Log.Debug("event accepted",
				elog.EventGroup(e),
			)
			c.Observe(metrics.EventAccepted, time.Since(now))
			now = time.Now()
		}
	}

	c.Log.Info(fmt.Sprintf("consumer for quene %v done, waiting for events delivery", c.queue.Name))

	c.exitCh <- struct{}{}
	<-c.doneCh

	c.Log.Info(fmt.Sprintf("consumer for quene %v closed", c.queue.Name))
}

type acker struct {
	acks map[uint64]*trackedDelivery

	queue            Queue
	ackSemaphore     chan struct{}
	exitIfQueueEmpty bool

	fetchCh chan trackedDelivery
	ackCh   chan uint64
	exitCh  chan struct{}
	doneCh  chan struct{}

	log *slog.Logger
}

func (a *acker) Run() {
	for {
		select {
		case msg := <-a.fetchCh: // message consumed
			a.log.Debug(fmt.Sprintf("accepted msg with delivery tag: %v from queue: %v", msg.DeliveryTag, a.queue.Name))
			a.acks[msg.DeliveryTag] = &msg
		case dTag := <-a.ackCh: // an event delivered
			a.log.Debug(fmt.Sprintf("got delivered tag: %v from queue: %v", dTag, a.queue.Name))
			delivered := false

			// mark message as delivered
			d, ok := a.acks[dTag]
			if ok {
				if d.events == 0 {
					delivered = true
				} else {
					d.events--
					if d.events == 0 {
						delivered = true
					}
				}
			} else { // normally it is never happens too
				a.log.Error("unexpected case, delivered tag not in map; please, report this issue")
			}

			if delivered {
				a.log.Debug(fmt.Sprintf("trying to ack tag: %v from queue: %v", dTag, a.queue.Name))
				if err := d.Ack(false); err != nil {
					a.log.Error(fmt.Sprintf("ack failed for tag: %v", dTag),
						msgLogAttrs(d.Delivery, "error", err, "queue", a.queue.Name)...,
					)
				} else {
					a.log.Debug(fmt.Sprintf("tag acked: %v from queue: %v", dTag, a.queue.Name))
				}

				delete(a.acks, dTag)
				_ = <-a.ackSemaphore //lint:ignore S1005 explicitly indicates reading from the channel, not waiting
			}

			if a.exitIfQueueEmpty && len(a.ackSemaphore) == 0 {
				close(a.doneCh)
				return
			}
		case <-a.exitCh: // consumer exited
			a.log.Info(fmt.Sprintf("left in ack queue: %v", len(a.ackSemaphore)))
			a.exitIfQueueEmpty = true
			if len(a.ackSemaphore) == 0 {
				close(a.doneCh)
				return
			}
		}
	}
}

func msgLogAttrs(msg amqp.Delivery, with ...any) []any {
	return append(with,
		"deliverytag", strconv.FormatUint(msg.DeliveryTag, 10),
		"consumertag", msg.ConsumerTag,
		"exchange", msg.Exchange,
		"routingkey", msg.RoutingKey,
		"redelivered", strconv.FormatBool(msg.Redelivered),
	)
}
