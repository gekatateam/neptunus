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

var (
	ErrNotACKed = errors.New("not ACKed by broker")
)

type eventPublishing struct {
	pub        amqp.Publishing
	event      *core.Event
	routingKey string
	err        error
	dur        time.Duration
}

type publisher struct {
	*core.BaseOutput

	exchange string
	input    chan *core.Event

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

func (p *publisher) Push(e *core.Event) {
	p.input <- e
}

func (p *publisher) Close() error {
	close(p.input)
	return nil
}

func (p *publisher) Run() {
	p.Log.Info(fmt.Sprintf("publisher for exchange %v spawned", p.exchange))

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

			headers := make(amqp.Table, len(p.headerLabels))
			for header, label := range p.headerLabels {
				if v, ok := e.GetLabel(label); ok {
					headers[header] = v
				}
			}

			pub := eventPublishing{
				event: e,
				err:   ErrNotACKed,
				pub: amqp.Publishing{
					DeliveryMode: p.dMode,
					AppId:        p.applicationId,
					Headers:      headers,
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

		now := time.Now()
		pubs, _ = p.produce(pubs)
		dur := durationPerEvent(time.Since(now), len(pubs))

		for _, pub := range pubs {
			p.Done <- pub.event
			if pub.err != nil {
				p.Log.Error("event produce failed",
					"error", pub.err,
					slog.Group("event",
						"id", pub.event.Id,
						"key", pub.event.RoutingKey,
					),
				)
				p.Observe(metrics.EventFailed, pub.dur+dur)
			} else {
				p.Log.Debug("event produced",
					slog.Group("event",
						"id", pub.event.Id,
						"key", pub.event.RoutingKey,
					),
				)
				p.Observe(metrics.EventAccepted, pub.dur+dur)
			}
		}
	})

	p.channel.Close()
	p.Log.Info(fmt.Sprintf("publisher for exchange %v closed", p.exchange))
}

func (p *publisher) produce(pubs []eventPublishing) ([]eventPublishing, error) {
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
		acks := make(map[uint64]int, len(pubs))

		for i, pub := range pubs {
			if pub.err != nil {
				acks[p.channel.GetNextPublishSeqNo()] = i

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
					p.Log.Error("message publish failed",
						"error", pub.err,
					)
				} else {
					pubs[i].err = nil
				}
			}
		}

		for c := range p.confirm {
			// normally, if acks is empty, it means than channel is already closed, but, you know
			// it is always better to check
			if len(acks) == 0 {
				break
			}

			if !c.Ack {
				pubs[acks[c.DeliveryTag]].err = ErrNotACKed
				hasErrorOrUnacked = true
				p.Log.Debug(fmt.Sprintf("unACKed tag: %v", c.DeliveryTag))
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

func durationPerEvent(totalTime time.Duration, batchSize int) time.Duration {
	return time.Duration(int64(totalTime) / int64(batchSize))
}
