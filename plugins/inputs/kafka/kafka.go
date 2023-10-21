package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
)

type Kafka struct {
	alias         string
	pipe          string
	EnableMetrics bool              `mapstructure:"enable_metrics"`
	Brokers       []string          `mapstructure:"brokers"`
	ClientId      string            `mapstructure:"client_id"`
	GroupId       string            `mapstructure:"group_id"`
	Topics        []string          `mapstructure:"topics"`
	Balancers     []string          `mapstructure:"balancers"`
	DialTimeout   time.Duration     `mapstructure:"dial_timeout"`
	SASL          SASL              `mapstructure:"sasl"`
	LabelHeaders  map[string]string `mapstructure:"labelheaders"`

	readersPool *readersPool
	configs map[string]*kafka.ReaderConfig
	reader *kafka.Reader
	fetchCtx context.Context

	log    *slog.Logger
	out    chan<- *core.Event
	parser core.Parser
}

type SASL struct {
	Mechanism string `mapstructure:"mechanism"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
}

func (i *Kafka) Init(config map[string]any, alias, pipeline string, log *slog.Logger) (err error) {
	if err = mapstructure.Decode(config, i); err != nil {
		return err
	}

	i.alias = alias
	i.pipe = pipeline
	i.log = log

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	if len(i.Brokers) == 0 {
		return errors.New("at least one broker address required")
	}

	if len(i.Topics) == 0 {
		return errors.New("at least one topic required")
	}

	if len(i.GroupId) == 0 {
		return errors.New("group_id required")
	}

	for _, topic := range i.Topics {
		var m sasl.Mechanism
		switch i.SASL.Mechanism {
		case "none":
		case "plain":
			m = &plain.Mechanism{
				Username: i.SASL.Username,
				Password: i.SASL.Password,
			}
		case "scram-sha-256":
			m, _ = scram.Mechanism(scram.SHA256, i.SASL.Username, i.SASL.Password)
		case "scram-sha-512":
			m, _ = scram.Mechanism(scram.SHA512, i.SASL.Username, i.SASL.Password)
		}

		readerConfig := &kafka.ReaderConfig{
			Brokers: i.Brokers,
			GroupID: i.GroupId,
			Topic:   topic,
			Dialer:  &kafka.Dialer{
				ClientID:  i.ClientId,
				DualStack: true,
				Timeout:   i.DialTimeout,
				SASLMechanism: m,
			},
			QueueCapacity: 1,
		}
	}


	return nil
}

func (i *Kafka) Prepare(out chan<- *core.Event) {
	i.out = out
}

func (i *Kafka) SetParser(p core.Parser) {
	i.parser = p
}

func (i *Kafka) Run() {
	for {
		msg, err := i.reader.FetchMessage(i.fetchCtx)
	}
}

func (i *Kafka) Close() error {
	return nil
}

func (i *Kafka) Alias() string {
	return i.alias
}

func init() {
	plugins.AddInput("kafka", func() core.Input {
		return &Kafka{

		}
	})
}
