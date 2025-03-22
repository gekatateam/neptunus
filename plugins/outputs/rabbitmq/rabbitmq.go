package rabbitmq

import (
	"errors"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	"github.com/gekatateam/neptunus/plugins/common/pool"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
	pkgtls "github.com/gekatateam/neptunus/plugins/common/tls"
)

type RabbitMQ struct {
	*core.BaseOutput  `mapstructure:"-"`
	Brokers           []string          `mapstructure:"brokers"`
	VHost             string            `mapstructure:"vhost"`
	ConnectionName    string            `mapstructure:"connection_name"`
	Username          string            `mapstructure:"username"`
	Password          string            `mapstructure:"password"`
	KeepTimestamp     bool              `mapstructure:"keep_timestamp"`
	KeepMessageId     bool              `mapstructure:"keep_message_id"`
	RoutingLabel      string            `mapstructure:"routing_label"`
	TypeLabel         string            `mapstructure:"type_label"`
	Mandatory         bool              `mapstructure:"mandatory"`
	Immediate         bool              `mapstructure:"immediate"`
	IdleTimeout       time.Duration     `mapstructure:"idle_timeout"`
	DialTimeout       time.Duration     `mapstructure:"dial_timeout"`
	HeartbeatInterval time.Duration     `mapstructure:"heartbeat_interval"`
	HeaderLabels      map[string]string `mapstructure:"headerlabels"`

	*pkgtls.TLSClientConfig       `mapstructure:",squash"`
	*batcher.Batcher[*core.Event] `mapstructure:",squash"`
	*retryer.Retryer              `mapstructure:",squash"`

	config amqp.Config
	conn   *amqp.Connection
	mu     *sync.Mutex

	producersPool *pool.Pool[*core.Event]
	ser           core.Serializer
}

func (o *RabbitMQ) Init() error {
	if len(o.Brokers) == 0 {
		return errors.New("at least one broker address required")
	}

	tlsConfig, err := o.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	o.config = amqp.Config{
		Vhost: o.VHost,
		SASL: []amqp.Authentication{
			&amqp.PlainAuth{
				Username: o.Username,
				Password: o.Password,
			},
		},
		Dial:            amqp.DefaultDial(o.DialTimeout),
		Heartbeat:       o.HeartbeatInterval,
		TLSClientConfig: tlsConfig,
	}
	panic("not implemented")
}

func (o *RabbitMQ) Close() error {
	o.ser.Close()
	return o.conn.Close()
}

func (o *RabbitMQ) Run() {
	clearTicker := time.NewTicker(time.Minute)
	if o.IdleTimeout == 0 {
		clearTicker.Stop()
	}

MAIN_LOOP:
	for {
		select {
		case e, ok := <-o.In:
			if !ok {
				clearTicker.Stop()
				break MAIN_LOOP
			}
			o.producersPool.Get(e.RoutingKey).Push(e)
		case <-clearTicker.C:
			for _, exchange := range o.producersPool.Keys() {
				if time.Since(o.producersPool.Get(exchange).LastWrite()) > o.IdleTimeout {
					o.producersPool.Remove(exchange)
				}
			}
		}
	}
}

func init() {
	plugins.AddOutput("rabbitmq", func() core.Output {
		return &RabbitMQ{
			VHost: "/",
			ConnectionName:    "neptunus.rabbitmq.output",
			DialTimeout:       10 * time.Second,
			HeartbeatInterval: 10 * time.Second,
			Batcher: &batcher.Batcher[*core.Event]{
				Buffer:   100,
				Interval: 5 * time.Second,
			},
			TLSClientConfig: &pkgtls.TLSClientConfig{},
			Retryer: &retryer.Retryer{
				RetryAttempts: 0,
				RetryAfter:    5 * time.Second,
			},
		}
	})
}
