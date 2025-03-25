package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/thanhpk/randstr"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/ider"
	pkgtls "github.com/gekatateam/neptunus/plugins/common/tls"
)

type RabbitMQ struct {
	*core.BaseInput   `mapstructure:"-"`
	Brokers           []string          `mapstructure:"brokers"`
	VHost             string            `mapstructure:"vhost"`
	ConsumerTag       string            `mapstructure:"consumer_tag"`
	ConnectionName    string            `mapstructure:"connection_name"`
	Username          string            `mapstructure:"username"`
	Password          string            `mapstructure:"password"`
	KeepTimestamp     bool              `mapstructure:"keep_timestamp"`
	KeepMessageId     bool              `mapstructure:"keep_message_id"`
	DialTimeout       time.Duration     `mapstructure:"dial_timeout"`
	HeartbeatInterval time.Duration     `mapstructure:"heartbeat_interval"`
	PrefetchCount     int               `mapstructure:"prefetch_count"`
	MaxUndelivered    int               `mapstructure:"max_undelivered"`
	Exchanges         []Exchange        `mapstructure:"exchanges"`
	Queues            []Queue           `mapstructure:"queues"`
	LabelHeaders      map[string]string `mapstructure:"labelheaders"`

	*ider.Ider              `mapstructure:",squash"`
	*pkgtls.TLSClientConfig `mapstructure:",squash"`

	cTag   string
	config amqp.Config
	conn   *amqp.Connection

	fetchCtx   context.Context
	cancelFunc context.CancelFunc
	doneCh     chan struct{}

	parser core.Parser
}

type Exchange struct {
	Name        string     `mapstructure:"name"`
	Type        string     `mapstructure:"type"`
	Durable     bool       `mapstructure:"durable"`
	AutoDelete  bool       `mapstructure:"auto_delete"`
	Passive     bool       `mapstructure:"passive"`
	DeclareArgs amqp.Table `mapstructure:"declare_args"`
}

type Queue struct {
	Name        string     `mapstructure:"name"`
	Durable     bool       `mapstructure:"durable"`
	AutoDelete  bool       `mapstructure:"auto_delete"` // if true, random suffix will be added to queue
	Exclusive   bool       `mapstructure:"exclusive"`   // if true, random suffix will be added to queue
	Requeue     bool       `mapstructure:"requeue"`
	Passive     bool       `mapstructure:"passive"`
	DeclareArgs amqp.Table `mapstructure:"declare_args"`
	ConsumeArgs amqp.Table `mapstructure:"consume_args"`
	Bindings    []Binding  `mapstructure:"bindings"`
}

type Binding struct {
	BindTo      string     `mapstructure:"bind_to"`
	BindingKey  string     `mapstructure:"binding_key"`
	DeclareArgs amqp.Table `mapstructure:"declare_args"`
}

func (i *RabbitMQ) Init() error {
	if len(i.Brokers) == 0 {
		return errors.New("at least one broker address required")
	}

	if len(i.Queues) == 0 {
		return errors.New("at least one queue required")
	}

	if err := i.Ider.Init(); err != nil {
		return err
	}

	tlsConfig, err := i.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	if err := i.ValidateDeclarations(); err != nil {
		return err
	}

	i.cTag = i.ConsumerTag + "-" + randstr.Base62(7)

	i.config = amqp.Config{
		Vhost: i.VHost,
		SASL: []amqp.Authentication{
			&amqp.PlainAuth{
				Username: i.Username,
				Password: i.Password,
			},
		},
		Dial:            amqp.DefaultDial(i.DialTimeout),
		Heartbeat:       i.HeartbeatInterval,
		TLSClientConfig: tlsConfig,
		Properties:      amqp.NewConnectionProperties(),
	}

	i.config.Properties.SetClientConnectionName(i.ConnectionName)
	i.fetchCtx, i.cancelFunc = context.WithCancel(context.Background())
	i.doneCh = make(chan struct{})

	return i.connect()
}

func (i *RabbitMQ) SetParser(p core.Parser) {
	i.parser = p
}

func (i *RabbitMQ) Close() error {
	i.cancelFunc()
	<-i.doneCh
	i.parser.Close()
	return i.conn.Close()
}

func (i *RabbitMQ) Run() {
	wg := &sync.WaitGroup{}

CONNECT_LOOP:
	for {
		select {
		case <-i.fetchCtx.Done():
			i.Log.Info("context cancelled, consumers closed")
			break CONNECT_LOOP
		default:
		}

		queues, err := i.declareAndBind()
		if err != nil {
			i.Log.Error("declarations failed",
				"error", err,
			)
			time.Sleep(i.DialTimeout)

			if err := i.connect(); err != nil {
				i.Log.Error("connection failed",
					"error", err,
				)
				continue CONNECT_LOOP
			}
		}

		for _, queue := range queues {
			ch, err := i.conn.Channel()
			if err != nil {
				i.Log.Error(fmt.Sprintf("cannot obtain amqp channel for queue: %v", queue.Name),
					"error", err,
				)
				continue
			}

			if err := ch.Qos(i.PrefetchCount, 0, false); err != nil {
				i.Log.Error(fmt.Sprintf("cannot set channel QoS for queue: %v", queue.Name),
					"error", err,
				)
				ch.Close()
				continue
			}

			deliveries, err := ch.ConsumeWithContext(
				i.fetchCtx,
				queue.Name,
				i.cTag,
				false, // autoAck
				queue.Exclusive,
				false, // noLocal
				false, // noWait
				queue.ConsumeArgs,
			)

			if err != nil {
				i.Log.Error(fmt.Sprintf("cannot obtain delivery chan for queue: %v", queue.Name),
					"error", err,
				)
				ch.Close()
				continue
			}

			var (
				ackSemaphore = make(chan struct{}, i.MaxUndelivered)
				fetchCh      = make(chan trackedDelivery)
				ackCh        = make(chan uint64)
				exitCh       = make(chan struct{})
				doneCh       = make(chan struct{})
			)

			consumer := &consumer{
				BaseInput:     i.BaseInput,
				keepTimestamp: i.KeepTimestamp,
				keepMessageId: i.KeepMessageId,
				labelHeaders:  i.LabelHeaders,
				ider:          i.Ider,
				parser:        i.parser,
				queue:         queue,
				input:         deliveries,
				ackSemaphore:  ackSemaphore,
				ackCh:         ackCh,
				fetchCh:       fetchCh,
				exitCh:        exitCh,
				doneCh:        doneCh,
			}

			acker := &acker{
				acks:             make(map[uint64]*trackedDelivery, i.MaxUndelivered),
				queue:            queue,
				ackSemaphore:     ackSemaphore,
				exitIfQueueEmpty: false,
				ackCh:            ackCh,
				fetchCh:          fetchCh,
				exitCh:           exitCh,
				doneCh:           doneCh,
				log:              i.Log,
			}

			wg.Add(2)
			go func() {
				defer ch.Close()
				defer wg.Done()
				consumer.Run()
			}()

			go func() {
				defer wg.Done()
				acker.Run()
			}()
		}
		wg.Wait() // this Wait() for CONNECT_LOOP blocking
	}
	wg.Wait()

	i.doneCh <- struct{}{}
}

func (i *RabbitMQ) ValidateDeclarations() error {
	exchanges := make(map[string]Exchange, len(i.Exchanges))
	for _, exchange := range i.Exchanges {
		if _, ok := exchanges[exchange.Name]; ok {
			return fmt.Errorf("exchange duplicate declaration: %v", exchange.Name)
		}

		switch exchange.Type {
		case amqp.ExchangeDirect, amqp.ExchangeFanout, amqp.ExchangeHeaders, amqp.ExchangeTopic:
		default:
			return fmt.Errorf("exchange %v unknowh type: %v; expected one of: %v, %v, %v, %v",
				exchange.Name, exchange.Type, amqp.ExchangeDirect, amqp.ExchangeFanout, amqp.ExchangeHeaders, amqp.ExchangeTopic)
		}

		if err := exchange.DeclareArgs.Validate(); err != nil {
			return fmt.Errorf("exchange %v declare args validation error: %w", exchange.Name, err)
		}
		exchanges[exchange.Name] = exchange
	}

	queues := make(map[string]Queue, len(i.Queues))
	for _, queue := range i.Queues {
		if _, ok := queues[queue.Name]; ok {
			return fmt.Errorf("queue duplicate declaration: %v", queue.Name)
		}

		if err := queue.DeclareArgs.Validate(); err != nil {
			return fmt.Errorf("queue %v declare args validation error: %w", queue.Name, err)
		}

		if err := queue.ConsumeArgs.Validate(); err != nil {
			return fmt.Errorf("queue %v consume args validation error: %w", queue.Name, err)
		}

		queues[queue.Name] = queue
		for _, binding := range queue.Bindings {
			if _, ok := exchanges[binding.BindTo]; !ok {
				return fmt.Errorf("found binding to undeclared exchange: %v; queue: %v, key: %v",
					binding.BindTo, queue.Name, binding.BindingKey)
			}

			if err := binding.DeclareArgs.Validate(); err != nil {
				return fmt.Errorf("queue %v binding to %v with key %v declare args validation error: %w",
					queue.Name, binding.BindTo, binding.BindingKey, err)
			}
		}
	}

	return nil
}

func (i *RabbitMQ) connect() error {
	conn, err := amqp.DialConfig(i.Brokers[rand.IntN(len(i.Brokers))], i.config)
	if err != nil {
		return err
	}

	i.conn = conn
	return nil
}

func (i *RabbitMQ) declareAndBind() ([]Queue, error) {
	ch, err := i.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("cannot obtain channel: %w", err)
	}
	defer ch.Close()

	for _, exchange := range i.Exchanges {
		var err error
		if exchange.Passive {
			err = ch.ExchangeDeclarePassive(
				exchange.Name,
				exchange.Type,
				exchange.Durable,
				exchange.AutoDelete,
				false, // internal
				false, // noWait
				exchange.DeclareArgs,
			)
		} else {
			err = ch.ExchangeDeclare(
				exchange.Name,
				exchange.Type,
				exchange.Durable,
				exchange.AutoDelete,
				false, // internal
				false, // noWait
				exchange.DeclareArgs,
			)
		}

		if err != nil {
			return nil, fmt.Errorf("exchange %v declaration failed: %w", exchange.Name, err)
		}

		i.Log.Debug(fmt.Sprintf("exchange %v declared", exchange.Name))
	}

	var consumeReadyQueues []Queue
	for _, queue := range i.Queues {
		if queue.Exclusive || queue.AutoDelete {
			queue.Name += "-" + randstr.Base62(7)
		}

		var q amqp.Queue
		var err error
		if queue.Passive {
			q, err = ch.QueueDeclarePassive(
				queue.Name,
				queue.Durable,
				queue.AutoDelete,
				queue.Exclusive,
				false, // noWait
				queue.DeclareArgs,
			)
		} else {
			q, err = ch.QueueDeclare(
				queue.Name,
				queue.Durable,
				queue.AutoDelete,
				queue.Exclusive,
				false, // noWait
				queue.DeclareArgs,
			)
		}

		if err != nil {
			return nil, fmt.Errorf("queue %v declaration failed: %w", queue.Name, err)
		}

		i.Log.Debug(fmt.Sprintf("queue %v declared", queue.Name))

		queue.Name = q.Name
		for _, binding := range queue.Bindings {
			err := ch.QueueBind(
				queue.Name,
				binding.BindingKey,
				binding.BindTo,
				false, // noWait
				binding.DeclareArgs,
			)
			if err != nil {
				return nil, fmt.Errorf("binding declaration %v to %v with key %v failed: %w",
					queue.Name, binding.BindTo, binding.BindingKey, err)
			}

			i.Log.Debug(fmt.Sprintf("binding %v to %v with key %v declared",
				queue.Name, binding.BindTo, binding.BindingKey))
		}

		consumeReadyQueues = append(consumeReadyQueues, queue)
	}

	return consumeReadyQueues, nil
}

func init() {
	plugins.AddInput("rabbitmq", func() core.Input {
		return &RabbitMQ{
			VHost:             "/",
			ConsumerTag:       "neptunus.rabbitmq.input",
			ConnectionName:    "neptunus.rabbitmq.input",
			DialTimeout:       10 * time.Second,
			HeartbeatInterval: 10 * time.Second,
			MaxUndelivered:    100,
			Ider:              &ider.Ider{},
			TLSClientConfig:   &pkgtls.TLSClientConfig{},
		}
	})
}
