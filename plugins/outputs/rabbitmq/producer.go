package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
)

type eventPublishing struct {
	pub        amqp.Publishing
	event      *core.Event
	routingKey string
	published  bool
	err        error
	dur        time.Duration
}

type producer struct {
	*core.BaseOutput

	input     chan *core.Event
	lastWrite time.Time

	keepTimestamp bool
	keepMessageId bool
	routingLabel  string
	typeLabel     string
	applicationId string
	dMode         uint8
	headerLabels  map[string]string

	ser         core.Serializer
	channelFunc func() (*amqp.Channel, error)
	channel     *amqp.Channel
	confirm     chan amqp.Confirmation

	*batcher.Batcher[*core.Event]
	*retryer.Retryer
}

func (p *producer) Push(e *core.Event) {
	p.input <- e
}

func (p *producer) LastWrite() time.Time {
	return p.lastWrite
}

func (p *producer) Close() error {
	close(p.input)
	return nil
}

func (p *producer) Run() {
	p.Batcher.Run(p.input, func(buf []*core.Event) {
		if len(buf) == 0 {
			return
		}

		pubs := make([]eventPublishing, 0, len(buf))
		for _, e := range buf {
			now := time.Now()
			event, err := p.ser.Serialize(e)
			if err != nil {
				p.Log.Error("serialization failed, event skipped",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				p.Done <- e
				p.Observe(metrics.EventFailed, time.Since(now))
				continue
			}

			pub := eventPublishing{
				event: e,
				pub: amqp.Publishing{
					DeliveryMode: p.dMode,
					AppId:        p.applicationId,
					Body:         event,
				},
			}

			if p.keepTimestamp {
				pub.pub.Timestamp = e.Timestamp
			}

			if p.keepMessageId {
				pub.pub.MessageId = e.Id
			}

			if len(p.typeLabel) > 0 {
				label, ok := e.GetLabel(p.typeLabel)
				if !ok {
					p.Log.Warn("event does not contains msgType label",
						slog.Group("event",
							"id", e.Id,
							"key", e.RoutingKey,
						),
					)
				} else {
					pub.pub.Type = label
				}
			}

			if len(p.routingLabel) > 0 {
				label, ok := e.GetLabel(p.routingLabel)
				if !ok {
					p.Log.Warn("event does not contains routingKey label",
						slog.Group("event",
							"id", e.Id,
							"key", e.RoutingKey,
						),
					)
				} else {
					pub.routingKey = label
				}
			}

			pub.dur = time.Since(now)
			pubs = append(pubs, pub)
		}

		if len(pubs) == 0 {
			p.Log.Warn("nothing to produce")
			return
		}

		err := p.produce(pubs)
	})
}

func (p *producer) produce(pubs []eventPublishing) ([]eventPublishing, error) {
	err := p.Retryer.Do("obtain channel and produce", p.Log, func() error {
		if p.channel == nil || p.channel.IsClosed() {
			channel, err := p.channelFunc()
			if err != nil {
				return fmt.Errorf("cannot obtain channel: %w", err)
			}

			if err := channel.Confirm(false); err != nil {
				return fmt.Errorf("cannot put channel into confirm mode: %w", err)
			}

			p.confirm = channel.NotifyPublish(make(chan amqp.Confirmation, p.Batcher.Buffer))
			p.channel = channel
		}

		hasErrorOrUnacked := false
		acks := make(map[uint64]*eventPublishing, len(pubs))

		for i, pub := range pubs {
			if !pub.published || pub.err != nil {
				acks[p.channel.GetNextPublishSeqNo()] = &pub

				err := p.channel.PublishWithContext(
					context.Background(), // context is not honoured
					pub.event.RoutingKey, // exchange
					pub.routingKey,       // amqp routingKey
					false,                // mandatory
					false,                // immediate
					pub.pub,
				)

				if err != nil {
					pubs[i].err = err
					hasErrorOrUnacked = true
				} else {
					pubs[i].err = nil
				}
			}
		}

		for c := range p.confirm {
			acks[c.DeliveryTag].published = c.Ack
			if !c.Ack {
				hasErrorOrUnacked = true
			}

			delete(acks, c.DeliveryTag)
			// channel will stay alive in most cases, so, we need to break loop if we got all confirms
			if len(acks) == 0 {
				break
			}
		}

		if hasErrorOrUnacked {
			return errors.New("not all messages published successfully")
		}

		return nil
	})

	return pubs, err
}
