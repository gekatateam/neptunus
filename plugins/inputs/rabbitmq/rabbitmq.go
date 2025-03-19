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
	DialTimeout       time.Duration     `mapstructure:"dial_timeout"`
	HeartbeatInterval time.Duration     `mapstructure:"heartbeat_interval"`
	PrefetchCount     int               `mapstructure:"prefetch_count"`
	MaxUndelivered    int               `mapstructure:"max_undelivered"`
	Exchanges         []Exchange        `mapstructure:"exchanges"`
	Queues            []Queue           `mapstructure:"queues"`
	LabelHeaders      map[string]string `mapstructure:"labelheaders"`

	*ider.Ider              `mapstructure:",squash"`
	*pkgtls.TLSClientConfig `mapstructure:",squash"`

	config amqp.Config
	conn   *amqp.Connection

	fetchCtx   context.Context
	cancelFunc context.CancelFunc

	parser core.Parser
}

type Exchange struct {
	Name        string            `mapstructure:"name"`
	Type        string            `mapstructure:"type"`
	Durable     bool              `mapstructure:"durable"`
	AutoDelete  bool              `mapstructure:"auto_delete"`
	DeclareArgs map[string]string `mapstructure:"declare_args"`

	declareArgs amqp.Table
}

type Queue struct {
	Name        string            `mapstructure:"name"`
	Durable     bool              `mapstructure:"durable"`
	AutoDelete  bool              `mapstructure:"auto_delete"` // if true, random suffix will be added to queue
	Exclusive   bool              `mapstructure:"exclusive"`   // if true, random suffix will be added to queue
	Requeue     bool              `mapstructure:"requeue"`
	DeclareArgs map[string]string `mapstructure:"declare_args"`
	ConsumeArgs map[string]string `mapstructure:"consume_args"`
	Bindings    []Binding         `mapstructure:"bindings"`

	declareArgs amqp.Table
	consumeArgs amqp.Table
}

type Binding struct {
	BindTo      string            `mapstructure:"bind_to"`
	BindingKey  string            `mapstructure:"binding_key"`
	DeclareArgs map[string]string `mapstructure:"declare_args"`

	declareArgs amqp.Table
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
	}

	i.fetchCtx, i.cancelFunc = context.WithCancel(context.Background())

	return i.connect()
}

func (i *RabbitMQ) SetParser(p core.Parser) {
	i.parser = p
}

func (i *RabbitMQ) Close() error {
	i.cancelFunc()
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
				i.ConsumerTag,
				false, // autoAck
				queue.Exclusive,
				false, // noLocal
				false, // noWait
				queue.consumeArgs,
			)

			if err != nil {
				i.Log.Error(fmt.Sprintf("cannot obtain delivery chan for queue: %v", queue.Name),
					"error", err,
				)
				ch.Close()
				continue
			}

			wg.Add(2)
			go func() {
				defer func() {
					ch.Close()
					wg.Done()
				}()
				// run consumer and acker
			}()
		}

		wg.Wait() // this Wait() for CONNECT_LOOP blocking
	}
	wg.Wait()
}

func (i *RabbitMQ) connect() error {
	conn, err := amqp.DialConfig(i.Brokers[rand.IntN(len(i.Brokers))], i.config)
	if err != nil {
		return err
	}

	i.conn = conn
	return nil
}

func (i *RabbitMQ) ValidateDeclarations() error {
	exchanges := make(map[string]Exchange, len(i.Exchanges))
	for j, exchange := range i.Exchanges {
		if _, ok := exchanges[exchange.Name]; ok {
			return fmt.Errorf("exchange duplicate declaration: %v", exchange.Name)
		}

		switch exchange.Type {
		case amqp.ExchangeDirect, amqp.ExchangeFanout, amqp.ExchangeHeaders, amqp.ExchangeTopic:
		default:
			return fmt.Errorf("exchange %v unknowh type: %v; expected one of: %v, %v, %v, %v",
				exchange.Name, exchange.Type, amqp.ExchangeDirect, amqp.ExchangeFanout, amqp.ExchangeHeaders, amqp.ExchangeTopic)
		}

		declareArgs := make(amqp.Table, len(exchange.DeclareArgs))
		for k, v := range exchange.DeclareArgs {
			declareArgs[k] = v
		}
		if err := declareArgs.Validate(); err != nil {
			return fmt.Errorf("exchange %v declare args validation error: %w", exchange.Name, err)
		}
		i.Exchanges[j].declareArgs = declareArgs

		exchanges[exchange.Name] = exchange
	}

	queues := make(map[string]Queue, len(i.Queues))
	for j, queue := range i.Queues {
		if _, ok := queues[queue.Name]; ok {
			return fmt.Errorf("queue duplicate declaration: %v", queue.Name)
		}

		declareArgs := make(amqp.Table, len(queue.DeclareArgs))
		for k, v := range queue.DeclareArgs {
			declareArgs[k] = v
		}
		if err := declareArgs.Validate(); err != nil {
			return fmt.Errorf("queue %v declare args validation error: %w", queue.Name, err)
		}
		i.Queues[j].declareArgs = declareArgs

		consumeArgs := make(amqp.Table, len(queue.ConsumeArgs))
		for k, v := range queue.ConsumeArgs {
			consumeArgs[k] = v
		}
		if err := consumeArgs.Validate(); err != nil {
			return fmt.Errorf("queue %v consume args validation error: %w", queue.Name, err)
		}
		i.Queues[j].consumeArgs = consumeArgs

		queues[queue.Name] = queue
		for k, binding := range queue.Bindings {
			if _, ok := exchanges[binding.BindTo]; !ok {
				return fmt.Errorf("found binding to undeclared exchange: %v; queue: %v, key: %v",
					binding.BindTo, queue.Name, binding.BindingKey)
			}

			declareArgs := make(amqp.Table, len(binding.DeclareArgs))
			for k, v := range binding.DeclareArgs {
				declareArgs[k] = v
			}
			if err := declareArgs.Validate(); err != nil {
				return fmt.Errorf("queue %v binding to %v with key %v declare args validation error: %w",
					queue.Name, binding.BindTo, binding.BindingKey)
			}
			i.Queues[j].Bindings[k].declareArgs = declareArgs
		}
	}

	return nil
}

func (i *RabbitMQ) declareAndBind() ([]Queue, error) {
	ch, err := i.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("cannot obtain channel: %w", err)
	}
	defer ch.Close()

	for _, exchange := range i.Exchanges {
		err := ch.ExchangeDeclare(
			exchange.Name,
			exchange.Type,
			exchange.Durable,
			exchange.AutoDelete,
			false, // internal
			false, // noWait
			exchange.declareArgs,
		)
		if err != nil {
			return nil, fmt.Errorf("exchange %v declaration failed: %w", exchange.Name, err)
		}
	}

	var consumeReadyQueues []Queue
	for _, queue := range i.Queues {
		if queue.Exclusive || queue.AutoDelete {
			queue.Name += randstr.Base62(7)
		}

		q, err := ch.QueueDeclare(
			queue.Name,
			queue.Durable,
			queue.AutoDelete,
			queue.Exclusive,
			false, // noWait
			queue.declareArgs,
		)
		if err != nil {
			return nil, fmt.Errorf("queue %v declaration failed: %w", queue.Name, err)
		}

		queue.Name = q.Name
		for _, binding := range queue.Bindings {
			err := ch.QueueBind(
				queue.Name,
				binding.BindingKey,
				binding.BindTo,
				false, // noWait
				binding.declareArgs,
			)
			if err != nil {
				return nil, fmt.Errorf("binding declaration %v to %v with key %v failed: %w",
					queue.Name, binding.BindTo, binding.BindingKey, err)
			}
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
			Ider:              &ider.Ider{},
			TLSClientConfig:   &pkgtls.TLSClientConfig{},
		}
	})
}
