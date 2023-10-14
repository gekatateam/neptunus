package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"
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
	id                uint64
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
	o.ClientId += strconv.Itoa(int(o.id))

	if err := mapstructure.Decode(config, o); err != nil {
		return err
	}

	o.alias = alias
	o.pipe = pipeline
	o.log = log

	if ok := checker.set(o.ClientId); !ok {
		return fmt.Errorf("duplicate ClientId found: %v", o.ClientId)
	}

	println("ID CHECK PASSED")

	if len(o.Brokers) == 0 {
		return errors.New("at least one broker address required")
	}

	if o.Batcher.Buffer < 0 {
		o.Batcher.Buffer = 1
	}

	writer := &kafka.Writer{
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
	default:
		return fmt.Errorf("unknown ack mode: %v; expected one of: none, one, all", o.RequiredAcks)
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
	case "none":
	case "plain":
		transport.SASL = &plain.Mechanism{
			Username: o.SASL.Username,
			Password: o.SASL.Password,
		}
	case "scram-sha-256":
		m, err := scram.Mechanism(scram.SHA256, o.SASL.Username, o.SASL.Password)
		if err != nil {
			return err
		}
		transport.SASL = m
	case "scram-sha-512":
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

func (o *Kafka) SetId(id uint64) {
	o.id = id
}

func (o *Kafka) Run() {
	o.Batcher.Run(o.in, func(buf []*core.Event) {
		if len(buf) == 0 {
			return
		}

		messages := []kafka.Message{}
		readyEvents := make(map[uuid.UUID]*eventMsgStatus)

		for _, e := range buf {
			now := time.Now()
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
				continue
			}

			msg := kafka.Message{
				Topic:      e.RoutingKey,
				Value:      event,
				WriterData: e.Id,
			}

			if o.KeepTimestamp {
				msg.Time = e.Timestamp
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
				if label, ok := e.GetLabel(o.PartitionLabel); ok {
					msg.Headers = append(msg.Headers, protocol.Header{
						Key:   o.PartitionLabel,
						Value: []byte(label),
					})
				}
			}

			if len(o.KeyLabel) > 0 {
				if label, ok := e.GetLabel(o.KeyLabel); ok {
					msg.Key = []byte(label)
				}
			}

			messages = append(messages, msg)
			readyEvents[e.Id] = &eventMsgStatus{
				event:     e,
				spentTime: time.Since(now),
				error:     nil,
			}
		}

		eventsStat := o.write(messages, readyEvents)

		// mark as done only after successful write
		// or when the maximum number of attempts has been reached
		for _, e := range eventsStat {
			e.event.Done()
			if e.error != nil {
				o.log.Error("event produce failed",
					"error", e.error,
					slog.Group("event",
						"id", e.event.Id,
						"key", e.event.RoutingKey,
					),
				)
				metrics.ObserveOutputSummary("kafka", o.alias, o.pipe, metrics.EventFailed, e.spentTime)
			} else {
				o.log.Debug("event produced",
					slog.Group("event",
						"id", e.event.Id,
						"key", e.event.RoutingKey,
					),
				)
				metrics.ObserveOutputSummary("kafka", o.alias, o.pipe, metrics.EventAccepted, e.spentTime)
			}
		}
	})
}

func (o *Kafka) Close() error {
	o.ser.Close()
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

func (o *Kafka) write(messages []kafka.Message, eventsStatus map[uuid.UUID]*eventMsgStatus) map[uuid.UUID]*eventMsgStatus {
	var attempts int = 0

SEND_LOOP:
	for {
		if len(messages) == 0 {
			return eventsStatus
		}

		now := time.Now()
		err := o.writer.WriteMessages(context.Background(), messages...)
		timePerEvent := durationPerEvent(time.Since(now), len(messages))

		switch writeErr := err.(type) {
		case nil: // all messages delivered successfully
			for _, m := range messages {
				successMsg := eventsStatus[m.WriterData.(uuid.UUID)]
				successMsg.spentTime += timePerEvent
				successMsg.error = nil
			}
			break SEND_LOOP
		case kafka.WriteErrors:
			var retriable []kafka.Message = nil
			for i, m := range messages {
				msgErr := writeErr[i]
				if msgErr == nil { // this message delivred successfully
					successMsg := eventsStatus[m.WriterData.(uuid.UUID)]
					successMsg.spentTime += timePerEvent
					successMsg.error = nil
					continue
				}

				if kafkaErr, ok := msgErr.(kafka.Error); ok {
					if kafkaErr.Timeout() || kafkaErr.Temporary() { // timeout and temporary errors are retriable
						retriable = append(retriable, m)
						retriableMsg := eventsStatus[m.WriterData.(uuid.UUID)]
						retriableMsg.spentTime += timePerEvent
						retriableMsg.error = kafkaErr
						continue
					}
				}

				// any other errors means than message cannot be produced
				failedMsg := eventsStatus[m.WriterData.(uuid.UUID)]
				failedMsg.spentTime += timePerEvent
				failedMsg.error = msgErr
			}
			messages = retriable
		case kafka.MessageTooLargeError: // exclude too large message and send others
			tooLargeMsg := eventsStatus[writeErr.Message.WriterData.(uuid.UUID)]
			tooLargeMsg.spentTime += timePerEvent
			tooLargeMsg.error = writeErr
			messages = writeErr.Remaining
		case kafka.Error:
			for _, m := range messages {
				kafkaMsg := eventsStatus[m.WriterData.(uuid.UUID)]
				kafkaMsg.spentTime += timePerEvent
				kafkaMsg.error = writeErr
			}

			if !(writeErr.Timeout() || writeErr.Temporary()) { // timeout and temporary errors are retriable
				break SEND_LOOP
			}
		default: // any other errors are unretriable
			for _, m := range messages {
				tooLargeMsg := eventsStatus[m.WriterData.(uuid.UUID)]
				tooLargeMsg.spentTime += timePerEvent
				tooLargeMsg.error = writeErr
			}
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

	return eventsStatus
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
			ClientId:          "neptunus.kafka.output.",
			DialTimeout:       5 * time.Second,
			WriteTimeout:      5 * time.Second,
			BatchTimeout:      100 * time.Millisecond,
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
