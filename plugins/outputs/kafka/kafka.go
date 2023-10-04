package log

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
	"github.com/segmentio/kafka-go/protocol"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
)

type Kafka struct {
	alias             string
	pipe              string
	EnableMetrics     bool              `mapstructure:"enable_metrics"`
	Brokers           []string          `mapstructure:"brokers"`
	DialTimeout       time.Duration     `mapstructure:"dial_timeout"`
	WriteTimeout      time.Duration     `mapstructure:"write_timeout"`
	BatchTimeout      time.Duration     `mapstructure:"batch_timeout"`
	TopicsAutocreate  bool              `mapstructure:"topics_autocreate"`
	Compression       string            `mapstructure:"compression"`
	MaxAttempts       int               `mapstructure:"max_attempts"`
	RetryAfter        time.Duration     `mapstructure:"retry_after"`
	PartitionBalancer string            `mapstructure:"partition_balancer"`
	PartitionLabel    string            `mapstructure:"partition_label"`
	SASL              SASL              `mapstructure:"sasl"`
	HeaderLabels      map[string]string `mapstructure:"headerlabels"`

	*batcher.Batcher[*core.Event] `mapstructure:",squash"`
	writer                        *kafka.Writer

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

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(o.Brokers...),
		RequiredAcks:           kafka.RequireOne,
		BatchSize:              o.Batcher.Buffer,
		AllowAutoTopicCreation: o.TopicsAutocreate,
		WriteTimeout:           o.WriteTimeout,
		BatchTimeout:           o.BatchTimeout,
		MaxAttempts:            1,
	}

	transport := &kafka.Transport{
		DialTimeout: o.DialTimeout,
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
		return fmt.Errorf("unknown compression algorithm: %v; expected one of: gzip, snappy, lz4, zstd", o.Compression)
	}

	switch o.SASL.Mechanism {
	case "plain":
		transport.SASL = &plain.Mechanism{
			Username: o.SASL.Username,
			Password: o.SASL.Password,
		}
	case "sha-256":
		m, err := scram.Mechanism(scram.SHA256, o.SASL.Username, o.SASL.Password)
		if err != nil {
			return err
		}
		transport.SASL = m
	case "sha-512":
		m, err := scram.Mechanism(scram.SHA512, o.SASL.Username, o.SASL.Password)
		if err != nil {
			return err
		}
		transport.SASL = m
	default:
		return fmt.Errorf("unknown SASL mechanism: %v; expected one of: plain, sha-256, sha-512", o.SASL.Mechanism)
	}

	switch o.PartitionBalancer {
	case "label":
		if len(o.PartitionLabel) == 0 {
			return errors.New("PartitionLabel requires for label balancer")
		}
		writer.Balancer = o
	case "round-robin":
		writer.Balancer = &kafka.RoundRobin{}
	case "least-bytes":
		writer.Balancer = &kafka.LeastBytes{}
	case "hash":
		writer.Balancer = &kafka.Hash{}
	case "reference-hash":
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
		return fmt.Errorf("unknown balancer: %v", o.PartitionBalancer)
	}

	client := &kafka.Client{
		Addr:      kafka.TCP(o.Brokers...),
		Transport: transport,
	}
	if _, err := client.ApiVersions(context.Background(), &kafka.ApiVersionsRequest{}); err != nil {
		return err
	}

	writer.Transport = transport
	o.writer = writer

	return nil
}

func (o *Kafka) Prepare(in <-chan *core.Event) {
	o.in = in
}

func (o *Kafka) SetSerializer(s core.Serializer) {
	o.ser = s
}

func (o *Kafka) Run() {
	o.Batcher.Run(o.in, func(buf []*core.Event) {
		now := time.Now()
		messages := []kafka.Message{}
		readyEvents := make([]*core.Event, 0, len(buf))

		for _, e := range buf {
			event, err := o.ser.Serialize(e)
			if err != nil {
				o.log.Error("serialization failed, event skipped",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				e.Done()
				metrics.ObserveOutputSummary("kafka", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
				now = time.Now()
				continue
			}

			msg := kafka.Message{
				Topic: e.RoutingKey,
				Time:  e.Timestamp,
				Value: event,
			}
			for header, label := range o.HeaderLabels {
				if v, ok := e.GetLabel(label); ok {
					msg.Headers = append(msg.Headers, protocol.Header{
						Key:   header,
						Value: []byte(v),
					})
				}
			}
			if o.PartitionBalancer == "label" {
				label, ok := e.GetLabel(o.PartitionLabel)
				if !ok {
					o.log.Error("label balancer configured, but event has no configured label",
						"error", err,
						slog.Group("event",
							"id", e.Id,
							"key", e.RoutingKey,
						),
					)
					e.Done()
					metrics.ObserveOutputSummary("kafka", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
					now = time.Now()
					continue
				}

				msg.Headers = append(msg.Headers, protocol.Header{
					Key:   o.PartitionLabel,
					Value: []byte(label),
				})
			}
			messages = append(messages, msg)
			readyEvents = append(readyEvents, e)
			now = time.Now()
		}

		var attempts int = 0
		var err error = nil
	SEND_LOOP:
		for {
			if err = o.writer.WriteMessages(context.Background(), messages...); err == nil {
				break SEND_LOOP
			}

			switch {
			case o.MaxAttempts > 0 && attempts < o.MaxAttempts:
				o.log.Warn(fmt.Sprintf("write %v of %v failed", attempts, o.MaxAttempts))
				attempts++
				time.Sleep(o.RetryAfter)
			case o.MaxAttempts > 0 && attempts >= o.MaxAttempts:
				o.log.Error(fmt.Sprintf("write failed after %v attemtps", attempts),
					"error", err,
				)
				break SEND_LOOP
			default:
				o.log.Error("write failed",
					"error", err,
				)
				time.Sleep(o.RetryAfter)
			}
		}

		for _, e := range readyEvents {
			// mark as done only after successful write
			// or when the maximum number of attempts has been reached
			e.Done()
			if err != nil {
				o.log.Error("event write failed",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				metrics.ObserveOutputSummary("kafka", o.alias, o.pipe, metrics.EventFailed, time.Since(now)) // THIS IS BAD MOVE
			} else {
				o.log.Debug("event sent",
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				metrics.ObserveOutputSummary("kafka", o.alias, o.pipe, metrics.EventAccepted, time.Since(now))
			}
		}
	})
}

func (o *Kafka) Close() error {
	return o.writer.Close()
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
				)
				return 0
			}

			if !slices.Contains(partitions, partition) {
				o.log.Warn("partition calculation error",
					"error", fmt.Sprintf("partition %v not in partitions list", partition),
				)
				return 0
			}

			return partition
		}
	}
	panic(fmt.Errorf("message has no %v header", o.PartitionLabel))
}

func init() {
	plugins.AddOutput("kafka", func() core.Output {
		return &Kafka{
			DialTimeout:       5 * time.Second,
			WriteTimeout:      5 * time.Second,
			BatchTimeout:      100 * time.Microsecond,
			RetryAfter:        5 * time.Second,
			Compression:       "none",
			PartitionBalancer: "least-bytes",
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
