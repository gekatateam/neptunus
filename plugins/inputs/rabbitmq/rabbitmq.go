package rabbitmq

import (
	"errors"
	"math/rand/v2"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

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
}

type Exchange struct {
	Name        string            `mapstructure:"name"`
	Type        string            `mapstructure:"type"`
	Durable     bool              `mapstructure:"durable"`
	AutoDelete  bool              `mapstructure:"auto_delete"`
	DeclareArgs map[string]string `mapstructure:"declare_args"`
}

type Queue struct {
	Name        string            `mapstructure:"name"`
	Durable     bool              `mapstructure:"durable"`
	AutoDelete  bool              `mapstructure:"auto_delete"`
	Exclusive   bool              `mapstructure:"exclusive"` // if true, random suffix will be added to queue
	Bindings    []Binding         `mapstructure:"bindings"`
	DeclareArgs map[string]string `mapstructure:"declare_args"`
	ConsumeArgs map[string]string `mapstructure:"consume_args"`
}

type Binding struct {
	Exchange    string            `mapstructure:"exchange"`
	BindingKey  string            `mapstructure:"binding_key"`
	DeclareArgs map[string]string `mapstructure:"declare_args"`
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

	i.config = amqp.Config{
		Vhost: i.VHost,
		SASL: []amqp.Authentication{
			&amqp.AMQPlainAuth{
				Username: i.Username,
				Password: i.Password,
			},
		},
		Dial:            amqp.DefaultDial(i.DialTimeout),
		Heartbeat:       i.HeartbeatInterval,
		TLSClientConfig: tlsConfig,
	}

	conn, err := amqp.DialConfig(i.Brokers[rand.IntN(len(i.Brokers))], i.config)
	if err != nil {
		return err
	}

	i.conn = conn
	return i.declareAndBind()
}

func (i *RabbitMQ) Close() error
func (i *RabbitMQ) Run()

func (i *RabbitMQ) declareAndBind() error {

}

func init() {
	plugins.AddInput("rabbitmq", func() core.Input {
		return &RabbitMQ{
			VHost:           "/",
			ConsumerTag:     "neptunus.rabbitmq.input",
			ConnectionName:  "neptunus.rabbitmq.input",
			DialTimeout:     10 * time.Second,
			Ider:            &ider.Ider{},
			TLSClientConfig: &pkgtls.TLSClientConfig{},
		}
	})
}
