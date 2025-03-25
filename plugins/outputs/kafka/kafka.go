package kafka

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"

	"kythe.io/kythe/go/util/datasize"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	common "github.com/gekatateam/neptunus/plugins/common/kafka"
	"github.com/gekatateam/neptunus/plugins/common/pool"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

type Kafka struct {
	*core.BaseOutput  `mapstructure:"-"`
	EnableMetrics     bool              `mapstructure:"enable_metrics"`
	Brokers           []string          `mapstructure:"brokers"`
	ClientId          string            `mapstructure:"client_id"`
	IdleTimeout       time.Duration     `mapstructure:"idle_timeout"`
	DialTimeout       time.Duration     `mapstructure:"dial_timeout"`
	WriteTimeout      time.Duration     `mapstructure:"write_timeout"`
	MaxMessageSize    datasize.Size     `mapstructure:"max_message_size"`
	TopicsAutocreate  bool              `mapstructure:"topics_autocreate"`
	Compression       string            `mapstructure:"compression"`
	RequiredAcks      string            `mapstructure:"required_acks"`
	KeepTimestamp     bool              `mapstructure:"keep_timestamp"`
	PartitionBalancer string            `mapstructure:"partition_balancer"`
	PartitionLabel    string            `mapstructure:"partition_label"`
	KeyLabel          string            `mapstructure:"key_label"`
	SASL              SASL              `mapstructure:"sasl"`
	HeaderLabels      map[string]string `mapstructure:"headerlabels"`

	*tls.TLSClientConfig          `mapstructure:",squash"`
	*batcher.Batcher[*core.Event] `mapstructure:",squash"`
	*retryer.Retryer              `mapstructure:",squash"`

	writersPool *pool.Pool[*core.Event]

	ser core.Serializer
}

type SASL struct {
	Mechanism string `mapstructure:"mechanism"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
}

func (o *Kafka) Init() error {
	if len(o.Brokers) == 0 {
		return errors.New("at least one broker address required")
	}

	if o.IdleTimeout > 0 && o.IdleTimeout < time.Minute {
		o.IdleTimeout = time.Minute
	}

	if o.Batcher.Buffer < 0 {
		o.Batcher.Buffer = 1
	}

	w, err := o.newWriter("")
	if err != nil {
		return err
	}

	defer w.Close()
	if err := o.testConn(w); err != nil {
		return err
	}

	o.writersPool = pool.New(func(topic string) pool.Runner[*core.Event] {
		return &topicWriter{
			BaseOutput:        o.BaseOutput,
			clientId:          o.ClientId,
			enableMetrics:     o.EnableMetrics,
			keepTimestamp:     o.KeepTimestamp,
			partitionBalancer: o.PartitionBalancer,
			partitionLabel:    o.PartitionLabel,
			keyLabel:          o.KeyLabel,
			headerLabels:      o.HeaderLabels,
			input:             make(chan *core.Event),
			writer:            func() *kafka.Writer { w, _ := o.newWriter(topic); return w }(),
			Batcher:           o.Batcher,
			Retryer:           o.Retryer,
			ser:               o.ser,
		}
	})

	return nil
}

func (o *Kafka) newWriter(topic string) (*kafka.Writer, error) {
	writer := &kafka.Writer{
		Topic:                  topic,
		Addr:                   kafka.TCP(o.Brokers...),
		BatchSize:              o.Batcher.Buffer,
		AllowAutoTopicCreation: o.TopicsAutocreate,
		WriteTimeout:           o.WriteTimeout,
		BatchTimeout:           time.Millisecond,
		BatchBytes:             int64(o.MaxMessageSize.Bytes()),
		MaxAttempts:            1,
		Logger:                 common.NewLogger(o.Log),
		ErrorLogger:            common.NewErrorLogger(o.Log),
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
		writer.Balancer = &labelBalancer{label: o.PartitionLabel, log: o.Log}
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

func (o *Kafka) SetSerializer(s core.Serializer) {
	o.ser = s
}

func (o *Kafka) Run() {
	clearTicker := time.NewTicker(time.Minute)
	if o.IdleTimeout == 0 {
		clearTicker.Stop()
	}

MAIN_LOOP:
	for {
		select {
		case e, ok := <-o.In:
			if !ok {
				clearTicker.Stop()
				break MAIN_LOOP
			}
			o.writersPool.Get(e.RoutingKey).Push(e)
		case <-clearTicker.C:
			for _, topic := range o.writersPool.Keys() {
				if time.Since(o.writersPool.LastWrite(topic)) > o.IdleTimeout {
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

func (o *Kafka) testConn(writer *kafka.Writer) error {
	dialer := &kafka.Dialer{
		ClientID:      writer.Transport.(*kafka.Transport).ClientID,
		DualStack:     true,
		Timeout:       writer.Transport.(*kafka.Transport).DialTimeout,
		SASLMechanism: writer.Transport.(*kafka.Transport).SASL,
		TLS:           writer.Transport.(*kafka.Transport).TLS,
	}

	ctx, cancel := context.WithTimeout(context.Background(), writer.Transport.(*kafka.Transport).DialTimeout)
	defer cancel()

	conn, err := dialer.DialContext(ctx, "tcp", o.Brokers[0])
	if err != nil {
		return err
	}

	conn.Close()
	return nil
}

func init() {
	plugins.AddOutput("kafka", func() core.Output {
		return &Kafka{
			ClientId:          "neptunus.kafka.output",
			IdleTimeout:       1 * time.Hour,
			DialTimeout:       5 * time.Second,
			WriteTimeout:      5 * time.Second,
			MaxMessageSize:    datasize.Mebibyte,
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
			Retryer: &retryer.Retryer{
				RetryAttempts: 0,
				RetryAfter:    5 * time.Second,
			},
		}
	})
}
