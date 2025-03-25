package rabbitmq

import (
	"errors"
	"fmt"
	"math/rand/v2"
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
	ApplicationId     string            `mapstructure:"application_id"`
	Username          string            `mapstructure:"username"`
	Password          string            `mapstructure:"password"`
	DeliveryMode      string            `mapstructure:"delivery_mode"`
	KeepTimestamp     bool              `mapstructure:"keep_timestamp"`
	KeepMessageId     bool              `mapstructure:"keep_message_id"`
	RoutingLabel      string            `mapstructure:"routing_label"`
	TypeLabel         string            `mapstructure:"type_label"`
	IdleTimeout       time.Duration     `mapstructure:"idle_timeout"`
	DialTimeout       time.Duration     `mapstructure:"dial_timeout"`
	HeartbeatInterval time.Duration     `mapstructure:"heartbeat_interval"`
	HeaderLabels      map[string]string `mapstructure:"headerlabels"`

	// TODO: add amqp.Return processing sometimes in future
	// Mandatory         bool              `mapstructure:"mandatory"`
	// Immediate         bool              `mapstructure:"immediate"`

	*pkgtls.TLSClientConfig       `mapstructure:",squash"`
	*batcher.Batcher[*core.Event] `mapstructure:",squash"`
	*retryer.Retryer              `mapstructure:",squash"`

	dMode  uint8
	config amqp.Config
	conn   *amqp.Connection
	mu     *sync.Mutex

	publishersPool *pool.Pool[*core.Event]
	ser            core.Serializer
}

func (o *RabbitMQ) Init() error {
	if len(o.Brokers) == 0 {
		return errors.New("at least one broker address required")
	}

	switch o.DeliveryMode {
	case "persistent":
		o.dMode = amqp.Persistent
	case "transient":
		o.dMode = amqp.Transient
	default:
		return fmt.Errorf("unknown delivery mode: %v; expected one of: persistent, transient", o.DeliveryMode)
	}

	tlsConfig, err := o.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	if o.IdleTimeout < time.Minute {
		o.IdleTimeout = time.Minute
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
		Properties:      amqp.NewConnectionProperties(),
	}

	o.config.Properties.SetClientConnectionName(o.ConnectionName)
	o.publishersPool = pool.New(o.newPublisher)
	o.mu = &sync.Mutex{}

	return o.connect()
}

func (o *RabbitMQ) SetSerializer(s core.Serializer) {
	o.ser = s
}

func (o *RabbitMQ) Close() error {
	o.ser.Close()
	o.publishersPool.Close()
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
			o.publishersPool.Get(e.RoutingKey).Push(e)
		case <-clearTicker.C:
			for _, exchange := range o.publishersPool.Keys() {
				if time.Since(o.publishersPool.LastWrite(exchange)) > o.IdleTimeout {
					o.publishersPool.Remove(exchange)
				}
			}
		}
	}
}

func (o *RabbitMQ) newPublisher(exchange string) pool.Runner[*core.Event] {
	return &publisher{
		BaseOutput:    o.BaseOutput,
		Retryer:       o.Retryer,
		Batcher:       o.Batcher,
		keepTimestamp: o.KeepTimestamp,
		keepMessageId: o.KeepMessageId,
		routingLabel:  o.RoutingLabel,
		typeLabel:     o.TypeLabel,
		applicationId: o.ApplicationId,
		dMode:         o.dMode,
		headerLabels:  o.HeaderLabels,
		ser:           o.ser,
		channelFunc:   o.channel,
		input:         make(chan *core.Event),
		exchange:      exchange,
	}
}

func (o *RabbitMQ) connect() error {
	conn, err := amqp.DialConfig(o.Brokers[rand.IntN(len(o.Brokers))], o.config)
	if err != nil {
		return err
	}

	o.conn = conn
	return nil
}

func (o *RabbitMQ) channel() (*amqp.Channel, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.conn.IsClosed() {
		if err := o.connect(); err != nil {
			return nil, err
		}
	}

	return o.conn.Channel()
}

func init() {
	plugins.AddOutput("rabbitmq", func() core.Output {
		return &RabbitMQ{
			VHost:             "/",
			ConnectionName:    "neptunus.rabbitmq.output",
			ApplicationId:     "neptunus.rabbitmq.output",
			DeliveryMode:      "persistent",
			IdleTimeout:       time.Hour,
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
