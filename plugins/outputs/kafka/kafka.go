package kafka

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
)

type Kafka struct {
	alias             string
	pipe              string
	EnableMetrics     bool              `mapstructure:"enable_metrics"`
	Brokers           []string          `mapstructure:"brokers"`
	ClientId          string            `mapstructure:"client_id"`
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

	*batcher.Batcher[*core.Event] `mapstructure:",squash"`

	writersPool *writersPool

	in  <-chan *core.Event
	log *slog.Logger
	ser core.Serializer
}

type SASL struct {
	Mechanism string `mapstructure:"mechanism"` // plain or scram
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

	if o.Batcher.Buffer < 0 {
		o.Batcher.Buffer = 1
	}

	switch o.RequiredAcks {
	case "none":
	case "one":
	case "all":
	default:
		return fmt.Errorf("unknown ack mode: %v; expected one of: none, one, all", o.RequiredAcks)
	}

	switch o.Compression {
	case "none":
	case "gzip":
	case "snappy":
	case "lz4":
	case "zstd":
	default:
		return fmt.Errorf("unknown compression algorithm: %v; expected one of: gzip, snappy, lz4, zstd", o.Compression)
	}

	switch o.SASL.Mechanism {
	case "none":
	case "plain":
	case "scram-sha-256":
		_, err := scram.Mechanism(scram.SHA256, o.SASL.Username, o.SASL.Password)
		if err != nil {
			return err
		}
	case "scram-sha-512":
		_, err := scram.Mechanism(scram.SHA512, o.SASL.Username, o.SASL.Password)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown SASL mechanism: %v; expected one of: plain, sha-256, sha-512", o.SASL.Mechanism)
	}

	switch o.PartitionBalancer {
	case "label":
		if len(o.PartitionLabel) == 0 {
			return errors.New("PartitionLabel requires for label balancer")
		}
	case "round-robin":
	case "least-bytes":
	case "fnv-1a":
	case "fnv-1a-reference":
	case "consistent-random":
	case "consistent":
	case "murmur2-random":
	case "murmur2":
	default:
		return fmt.Errorf("unknown balancer: %v", o.PartitionBalancer)
	}

	o.writersPool = &writersPool{
		writers: make(map[string]*topicWriter),
		mu:      &sync.Mutex{},
		wg:      &sync.WaitGroup{},
		new: func(topic string) *topicWriter {
			writer := o.newWriter(topic)
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
				writer:            writer,
				batcher:           o.Batcher,
				log:               o.log,
				ser:               o.ser,
			}
		},
	}

	return nil
}

func (o *Kafka) Prepare(in <-chan *core.Event) {
	o.in = in
}

func (o *Kafka) SetSerializer(s core.Serializer) {
	o.ser = s
}

func (o *Kafka) Run() {
	for e := range o.in {
		o.writersPool.Get(e.RoutingKey) <- e
	}

	o.writersPool.Close()
}

func (o *Kafka) Close() error {
	o.ser.Close()
	return nil
}

func (o *Kafka) Alias() string {
	return o.alias
}

func (o *Kafka) Balance(msg kafka.Message, partitions ...int) (partition int) {
	for _, h := range msg.Headers {
		if h.Key == o.PartitionLabel {
			partition, err := strconv.Atoi(string(h.Value))
			if err != nil {
				o.log.Warn("partition calculation error",
					"error", err,
					slog.Group("event",
						"id", msg.WriterData.(uuid.UUID),
						"key", msg.Topic,
					),
				)
				return partitions[0]
			}

			if !slices.Contains(partitions, partition) {
				o.log.Warn("partition calculation error",
					"error", fmt.Sprintf("partition %v not in partitions list", partition),
					slog.Group("event",
						"id", msg.WriterData.(uuid.UUID),
						"key", msg.Topic,
					),
				)
				return partitions[0]
			}

			return partition
		}
	}
	o.log.Warn("partition calculation error",
		"error", fmt.Sprintf("message has no %v header, 0 partition used", o.PartitionLabel),
		slog.Group("event",
			"id", msg.WriterData.(uuid.UUID),
			"key", msg.Topic,
		),
	)
	return partitions[0]
}

func (o *Kafka) newWriter(topic string) *kafka.Writer {
	writer := &kafka.Writer{
		Topic:                  topic,
		Addr:                   kafka.TCP(o.Brokers...),
		BatchSize:              o.Batcher.Buffer,
		AllowAutoTopicCreation: o.TopicsAutocreate,
		WriteTimeout:           o.WriteTimeout,
		BatchTimeout:           o.BatchTimeout,
		BatchBytes:             o.MaxMessageSize,
		MaxAttempts:            1,
	}

	transport := &kafka.Transport{
		DialTimeout: o.DialTimeout,
		ClientID:    o.ClientId,
	}

	switch o.RequiredAcks {
	case "none":
		writer.RequiredAcks = kafka.RequireNone
	case "one":
		writer.RequiredAcks = kafka.RequireOne
	case "all":
		writer.RequiredAcks = kafka.RequireAll
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
	}

	switch o.SASL.Mechanism {
	case "none":
	case "plain":
		transport.SASL = &plain.Mechanism{
			Username: o.SASL.Username,
			Password: o.SASL.Password,
		}
	case "scram-sha-256":
		m, _ := scram.Mechanism(scram.SHA256, o.SASL.Username, o.SASL.Password)
		transport.SASL = m
	case "scram-sha-512":
		m, _ := scram.Mechanism(scram.SHA512, o.SASL.Username, o.SASL.Password)
		transport.SASL = m
	}

	switch o.PartitionBalancer {
	case "label":
		writer.Balancer = o
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
	}

	writer.Transport = transport
	return writer
}

type eventMsgStatus struct {
	event     *core.Event
	spentTime time.Duration
	error     error
}

func durationPerEvent(totalTime time.Duration, batchSize int) time.Duration {
	return time.Duration(int64(totalTime) / int64(batchSize))
}

func init() {
	plugins.AddOutput("kafka", func() core.Output {
		return &Kafka{
			ClientId:          "neptunus.kafka.output",
			DialTimeout:       5 * time.Second,
			WriteTimeout:      5 * time.Second,
			BatchTimeout:      10 * time.Millisecond,
			MaxMessageSize:    1_048_576, // 1 MiB
			RetryAfter:        5 * time.Second,
			RequiredAcks:      "one",
			Compression:       "none",
			PartitionBalancer: "least-bytes",
			Batcher: &batcher.Batcher[*core.Event]{
				Buffer:   100,
				Interval: 5 * time.Second,
			},
			SASL: SASL{
				Mechanism: "scram-sha-512",
			},
		}
	})
}
