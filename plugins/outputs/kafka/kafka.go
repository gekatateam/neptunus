package log

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
)

type Kafka struct {
	alias            string
	pipe             string
	EnableMetrics    bool              `mapstructure:"enable_metrics"`
	Brokers          []string          `mapstructure:"brokers"`
	DialTimeout      time.Duration     `mapstructure:"dial_timeout"`
	TopicsAutocreate bool              `mapstructure:"topics_autocreate"`
	Compression      string            `mapstructure:"compression"` // gzip, snappy, lz4 or zstd
	MaxAttempts      int               `mapstructure:"max_attempts"`
	PartitionLabel   string            `mapstructure:"partition_label"`
	SASL             SASL              `mapstructure:"sasl"`
	HeaderLabels     map[string]string `mapstructure:"headerlabels"`

	*batcher.Batcher[*core.Event]
	writer *kafka.Writer

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

	writerConfig := kafka.WriterConfig{
		Dialer: &kafka.Dialer{
			Timeout:   o.DialTimeout,
			DualStack: true,
		},
		Brokers:     o.Brokers,
		MaxAttempts: o.MaxAttempts,
		BatchSize:   o.Batcher.Buffer,
	}

	// configure Writer retries
	// zero means infinite number of retries that will be executed in the Run() loop
	if o.MaxAttempts == 0 {
		writerConfig.MaxAttempts = 10
	}

	switch o.Compression {
	case "gzip":
		writerConfig.CompressionCodec = &compress.GzipCodec
	case "snappy":
		writerConfig.CompressionCodec = &compress.SnappyCodec
	case "lz4":
		writerConfig.CompressionCodec = &compress.Lz4Codec
	case "zstd":
		writerConfig.CompressionCodec = &compress.ZstdCodec
	default:
		return fmt.Errorf("unknown compression algorithm: %v; expected one of: gzip, snappy, lz4, zstd", o.Compression)
	}

	switch o.SASL.Mechanism {
	case "plain":
		writerConfig.Dialer.SASLMechanism = &plain.Mechanism{
			Username: o.SASL.Username,
			Password: o.SASL.Password,
		}
	case "scram":
		m, err := scram.Mechanism(scram.SHA512, o.SASL.Username, o.SASL.Password)
		if err != nil {
			return err
		}
		writerConfig.Dialer.SASLMechanism = m
	default:
		return fmt.Errorf("unknown SASL mechanism: %v; expected one of: plain, scram", o.SASL.Mechanism)
	}

	if err := writerConfig.Validate(); err != nil {
		return err
	}

	o.writer = kafka.NewWriter(writerConfig)

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
		now := time.Now()
		event, err := o.ser.Serialize(e)
		if err != nil {
			o.log.Error("serialization failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.Done()
			metrics.ObserveOutputSummary("log", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
			continue
		}

		fmt.Println(event)

		e.Done()
		metrics.ObserveOutputSummary("log", o.alias, o.pipe, metrics.EventAccepted, time.Since(now))
	}
}

func (o *Kafka) Close() error {
	return o.writer.Close()
}

func (o *Kafka) Alias() string {
	return o.alias
}

func init() {
	plugins.AddOutput("kafka", func() core.Output {
		return &Kafka{
			DialTimeout: 5 * time.Second,
			Compression: "gzip",
			Batcher: &batcher.Batcher[*core.Event]{
				Buffer:   100,
				Interval: 5 * time.Second,
			},
			SASL: SASL{
				Mechanism: "scram",
			},
		}
	})
}
