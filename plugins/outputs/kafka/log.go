package log

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
)

type Kafka struct {
	alias string
	pipe  string
	EnableMetrics bool `mapstructure:"enable_metrics"`
	Brokers []string `mapstructure:"brokers"`
	Compression string `mapstructure:"compression"` // gzip, snappy, lz4 or zstd
	MaxAttempts int `mapstructure:"max_attempts"`
	PartitionLabel string `mapstructure:"partition_label"`
	SASL SASL `mapstructure:"sasl"`

	*batcher.Batcher[*core.Event]
	writer *kafka.Writer

	in  <-chan *core.Event
	log *slog.Logger
	ser core.Serializer
}

type SASL struct {
	Mechanism string `mapstructure:"mechanism"` // plain or scram
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

func (o *Kafka) Init(config map[string]any, alias, pipeline string, log *slog.Logger) error {
	if err := mapstructure.Decode(config, o); err != nil {
		return err
	}

	o.alias = alias
	o.pipe = pipeline
	o.log = log



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
	return nil
}

func (o *Kafka) Alias() string {
	return o.alias
}

func writerConfig(cfg *Kafka) (*kafka.WriterConfig, error) {

}

func saslConfig(cfg *SASL) (sasl.Mechanism, error) {
	switch cfg.Mechanism {
	case "plain":
		return &plain.Mechanism{
			Username: cfg.Username,
			Password: cfg.Password,
		}, nil
	case "scram":
		return scram.Mechanism(scram.SHA512, cfg.Username, cfg.Password)
	default:
		return nil, fmt.Errorf("unknown SASL mechanism: %v; expected one of: plain, scram", cfg.Mechanism)
	}
}

func init() {
	plugins.AddOutput("kafka", func() core.Output {
		return &Kafka{
			
		}
	})
}
