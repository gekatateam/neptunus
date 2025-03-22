package rabbitmq

import (
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
)

type producer struct {
	*core.BaseOutput

	input     chan *core.Event
	lastWrite time.Time

	keepTimestamp bool
	keepMessageId bool
	routingLabel  string
	typeLabel     string
	mandatory     bool
	immediate     bool
	dMode         uint8
	headerLabels  map[string]string

	ser         core.Serializer
	channelFunc func() (*amqp.Channel, error)
	channel     *amqp.Channel

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
		now := time.Now()

		pubs := make([]amqp.Publishing, 0, len(buf))
		for _, e := range buf {
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
		}











		
	})
}
