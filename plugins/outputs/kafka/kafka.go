package kafka

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	common "github.com/gekatateam/neptunus/plugins/common/kafka"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

type Kafka struct {
	alias             string
	pipe              string
	EnableMetrics     bool              `mapstructure:"enable_metrics"`
	Brokers           []string          `mapstructure:"brokers"`
	ClientId          string            `mapstructure:"client_id"`
	IdleTimeout       time.Duration     `mapstructure:"idle_timeout"`
	DialTimeout       time.Duration     `mapstructure:"dial_timeout"`
	WriteTimeout      time.Duration     `mapstructure:"write_timeout"`
	BatchTimeout      time.Duration     `mapstructure:"batch_timeout"`
	MaxMessageSize    int64             `mapstructure:"max_message_size"`
	TopicsAutocreate  bool              `mapstructure:"topics_autocreate"`
	Compression       string            `mapstructure:"compression"`
	RequiredAcks      string            `mapstructure:"required_acks"`
	KeepTimestamp     bool              `mapstructure:"keep_timestamp"`
	PartitionBalancer string            `mapstructure:"partition_balancer"`
	PartitionLabel    string            `mapstructure:"partition_label"`
	KeyLabel          string            `mapstructure:"key_label"`
	MaxAttempts       int               `mapstructure:"max_attempts"`
	RetryAfter        time.Duration     `mapstructure:"retry_after"`
	SASL              SASL              `mapstructure:"sasl"`
	HeaderLabels      map[string]string `mapstructure:"headerlabels"`

	*tls.TLSClientConfig          `mapstructure:",squash"`
	*batcher.Batcher[*core.Event] `mapstructure:",squash"`

	writersPool *writersPool
	clearTicker *time.Ticker

	in  <-chan *core.Event
	log *slog.Logger
	ser core.Serializer
}

type SASL struct {
	Mechanism string `mapstructure:"mechanism"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
}

func (o *Kafka) Init(config map[string]any, alias, pipeline string, log *slog.Logger) error {
	if err := mapstructure.Decode(config, o); err != nil {
		return err
	}

	o.alias = alias
	o.pipe = pipeline
	o.log = log

	if len(o.Brokers) == 0 {
		return errors.New("at least one broker address required")
	}

	if o.IdleTimeout > 0 && o.IdleTimeout < time.Minute {
		o.IdleTimeout = time.Minute
	}

	if o.Batcher.Buffer < 0 {
		o.Batcher.Buffer = 1
	}

	if _, err := o.newWriter(""); err != nil {
		return err
	}

	o.writersPool = &writersPool{
		writers: make(map[string]*topicWriter),
		wg:      &sync.WaitGroup{},
		new: func(topic string) *topicWriter {
			return &topicWriter{
				alias:             o.alias,
				pipe:              o.pipe,
				clientId:          o.ClientId,
				enableMetrics:     o.EnableMetrics,
				keepTimestamp:     o.KeepTimestamp,
				partitionBalancer: o.PartitionBalancer,
				partitionLabel:    o.PartitionLabel,
				keyLabel:          o.KeyLabel,
				headerLabels:      o.HeaderLabels,
				maxAttempts:       o.MaxAttempts,
				retryAfter:        o.RetryAfter,
				input:             make(chan *core.Event),
				writer:            func() *kafka.Writer { w, _ := o.newWriter(topic); return w }(),
				batcher:           o.Batcher,
				log:               o.log,
				ser:               o.ser,
			}
		},
	}

	o.clearTicker = time.NewTicker(time.Minute)
	if o.IdleTimeout == 0 {
		o.clearTicker.Stop()
	}

	return nil
}

func (o *Kafka) newWriter(topic string) (*kafka.Writer, error) {
	writer := &kafka.Writer{
		Topic:                  topic,
		Addr:                   kafka.TCP(o.Brokers...),
		BatchSize:              o.Batcher.Buffer,
		AllowAutoTopicCreation: o.TopicsAutocreate,
		WriteTimeout:           o.WriteTimeout,
		BatchTimeout:           o.BatchTimeout,
		BatchBytes:             o.MaxMessageSize,
		MaxAttempts:            1,
		Logger:                 common.NewLogger(o.log),
		ErrorLogger:            common.NewErrorLogger(o.log),
	}

	transport := &kafka.Transport{
		DialTimeout: o.DialTimeout,
		ClientID:    o.ClientId,
	}

	tlsConfig, err := o.TLSClientConfig.Config()
	if err != nil {
		return nil, err
	}
	transport.TLS = tlsConfig

	switch o.RequiredAcks {
	case "none":
		writer.RequiredAcks = kafka.RequireNone
	case "one":
		writer.RequiredAcks = kafka.RequireOne
	case "all":
		writer.RequiredAcks = kafka.RequireAll
	default:
		return nil, fmt.Errorf("unknown ack mode: %v; expected one of: none, one, all", o.RequiredAcks)
	}

	switch o.Compression {
	case "none":
		writer.Compression = compress.None
	case "gzip":
		writer.Compression = compress.Gzip
	case "snappy":
		writer.Compression = compress.Snappy
	case "lz4":
		writer.Compression = compress.Lz4
	case "zstd":
		writer.Compression = compress.Zstd
	default:
		return nil, fmt.Errorf("unknown compression algorithm: %v; expected one of: gzip, snappy, lz4, zstd", o.Compression)
	}

	switch o.SASL.Mechanism {
	case "none":
	case "plain":
		transport.SASL = &plain.Mechanism{
			Username: o.SASL.Username,
			Password: o.SASL.Password,
		}
	case "scram-sha-256":
		m, err := scram.Mechanism(scram.SHA256, o.SASL.Username, o.SASL.Password)
		if err != nil {
			return nil, err
		}
		transport.SASL = m
	case "scram-sha-512":
		m, err := scram.Mechanism(scram.SHA512, o.SASL.Username, o.SASL.Password)
		if err != nil {
			return nil, err
		}
		transport.SASL = m
	default:
		return nil, fmt.Errorf("unknown SASL mechanism: %v; expected one of: plain, sha-256, sha-512", o.SASL.Mechanism)
	}

	switch o.PartitionBalancer {
	case "label":
		if len(o.PartitionLabel) == 0 {
			return nil, errors.New("PartitionLabel requires for label balancer")
		}
		writer.Balancer = &labelBalancer{label: o.PartitionLabel, log: o.log}
	case "round-robin":
		writer.Balancer = &kafka.RoundRobin{}
	case "least-bytes":
		writer.Balancer = &kafka.LeastBytes{}
	case "fnv-1a":
		writer.Balancer = &kafka.Hash{}
	case "fnv-1a-reference":
		writer.Balancer = &kafka.ReferenceHash{}
	case "consistent-random":
		writer.Balancer = &kafka.CRC32Balancer{}
	case "consistent":
		writer.Balancer = &kafka.CRC32Balancer{Consistent: true}
	case "murmur2-random":
		writer.Balancer = &kafka.Murmur2Balancer{}
	case "murmur2":
		writer.Balancer = &kafka.Murmur2Balancer{Consistent: true}
	default:
		return nil, fmt.Errorf("unknown balancer: %v", o.PartitionBalancer)
	}

	writer.Transport = transport
	return writer, nil
}

func (o *Kafka) Prepare(in <-chan *core.Event) {
	o.in = in
}

func (o *Kafka) SetSerializer(s core.Serializer) {
	o.ser = s
}

func (o *Kafka) Run() {
MAIN_LOOP:
	for {
		select {
		case e, ok := <-o.in:
			if !ok {
				o.clearTicker.Stop()
				break MAIN_LOOP
			}
			o.writersPool.Get(e.RoutingKey).input <- e
		case <-o.clearTicker.C:
			for _, topic := range o.writersPool.Topics() {
				if time.Since(o.writersPool.Get(topic).lastWrite) > o.IdleTimeout {
					o.writersPool.Remove(topic)
				}
			}
		}
	}
}

func (o *Kafka) Close() error {
	o.writersPool.Close()
	o.ser.Close()
	return nil
}

func (o *Kafka) Alias() string {
	return o.alias
}

func init() {
	plugins.AddOutput("kafka", func() core.Output {
		return &Kafka{
			ClientId:          "neptunus.kafka.output",
			IdleTimeout:       1 * time.Hour,
			DialTimeout:       5 * time.Second,
			WriteTimeout:      5 * time.Second,
			BatchTimeout:      10 * time.Millisecond,
			MaxMessageSize:    1_048_576, // 1 MiB
			RetryAfter:        5 * time.Second,
			RequiredAcks:      "one",
			Compression:       "none",
			PartitionBalancer: "least-bytes",
			SASL: SASL{
				Mechanism: "scram-sha-512",
			},
			Batcher: &batcher.Batcher[*core.Event]{
				Buffer:   100,
				Interval: 5 * time.Second,
			},
			TLSClientConfig: &tls.TLSClientConfig{},
		}
	})
}
