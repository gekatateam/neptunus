package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
	orderedmap "github.com/wk8/go-ordered-map/v2"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/ider"
	common "github.com/gekatateam/neptunus/plugins/common/kafka"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

type Kafka struct {
	alias                string
	pipe                 string
	EnableMetrics        bool              `mapstructure:"enable_metrics"`
	Brokers              []string          `mapstructure:"brokers"`
	ClientId             string            `mapstructure:"client_id"`
	GroupId              string            `mapstructure:"group_id"`
	GroupTTL             time.Duration     `mapstructure:"group_ttl"`
	GroupBalancer        string            `mapstructure:"group_balancer"`
	Rack                 string            `mapstructure:"rack"`
	Topics               []string          `mapstructure:"topics"`
	DialTimeout          time.Duration     `mapstructure:"dial_timeout"`
	SessionTimeout       time.Duration     `mapstructure:"session_timeout"`
	RebalanceTimeout     time.Duration     `mapstructure:"rebalance_timeout"`
	HeartbeatInterval    time.Duration     `mapstructure:"heartbeat_interval"`
	ReadBatchTimeout     time.Duration     `mapstructure:"read_batch_timeout"`
	WaitBatchTimeout     time.Duration     `mapstructure:"wait_batch_timeout"`
	StartOffset          string            `mapstructure:"start_offset"`
	MaxBatchSize         int               `mapstructure:"max_batch_size"`
	MaxUncommitted       int               `mapstructure:"max_uncommitted"`
	CommitInterval       time.Duration     `mapstructure:"commit_interval"`
	SASL                 SASL              `mapstructure:"sasl"`
	LabelHeaders         map[string]string `mapstructure:"labelheaders"`
	*ider.Ider           `mapstructure:",squash"`
	*tls.TLSClientConfig `mapstructure:",squash"`

	readersPool    map[string]*topicReader
	commitConsPool map[string]*commitController
	fetchCtx       context.Context
	cancelFunc     context.CancelFunc
	wg             *sync.WaitGroup

	log    *slog.Logger
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
			for _, v := range i.readersPool {
				v.reader.Close()
			}
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

	if err := i.Ider.Init(); err != nil {
		return err
	}

	if i.MaxUncommitted < 0 {
		i.MaxUncommitted = 100
	}

	i.Topics = slices.Compact(i.Topics)
	i.readersPool = make(map[string]*topicReader)
	i.commitConsPool = make(map[string]*commitController)
	i.wg = &sync.WaitGroup{}

	for _, topic := range i.Topics {
		var m sasl.Mechanism
		var err error
		switch i.SASL.Mechanism {
		case "none":
		case "plain":
			m = &plain.Mechanism{
				Username: i.SASL.Username,
				Password: i.SASL.Password,
			}
		case "scram-sha-256":
			m, err = scram.Mechanism(scram.SHA256, i.SASL.Username, i.SASL.Password)
			if err != nil {
				return err
			}
		case "scram-sha-512":
			m, err = scram.Mechanism(scram.SHA512, i.SASL.Username, i.SASL.Password)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown SASL mechanism: %v; expected one of: plain, sha-256, sha-512", i.SASL.Mechanism)
		}

		var offset int64
		switch i.StartOffset {
		case "first":
			offset = kafka.FirstOffset
		case "last":
			offset = kafka.LastOffset
		default:
			return fmt.Errorf("unknown offset: %v; expected one of: first, last", i.StartOffset)
		}

		var groupBalancer kafka.GroupBalancer
		switch i.GroupBalancer {
		case "range":
			groupBalancer = &kafka.RangeGroupBalancer{}
		case "round-robin":
			groupBalancer = &kafka.RoundRobinGroupBalancer{}
		case "rack-affinity":
			if len(i.Rack) == 0 {
				return errors.New("rack required for rack-affinity gorup balancer")
			}
			groupBalancer = &kafka.RackAffinityGroupBalancer{
				Rack: i.Rack,
			}
		default:
			return fmt.Errorf("unknown group balancer: %v; expected one of: range, round-robin, rack-affinity", i.GroupBalancer)
		}

		tlsConfig, err := i.TLSClientConfig.Config()
		if err != nil {
			return err
		}

		var (
			fetchCh  = make(chan kafka.Message)
			commitCh = make(chan int64)
			exitCh   = make(chan struct{})
			doneCh   = make(chan struct{})
			semCh    = make(chan struct{}, i.MaxUncommitted)
		)

		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers: i.Brokers,
			GroupID: i.GroupId,
			Topic:   topic,
			Dialer: &kafka.Dialer{
				ClientID:      i.ClientId,
				DualStack:     true,
				Timeout:       i.DialTimeout,
				SASLMechanism: m,
				TLS:           tlsConfig,
			},
			MaxBytes:              i.MaxBatchSize,
			QueueCapacity:         1,
			MaxAttempts:           1,
			WatchPartitionChanges: true,
			StartOffset:           offset,
			HeartbeatInterval:     i.HeartbeatInterval,
			SessionTimeout:        i.SessionTimeout,
			RebalanceTimeout:      i.RebalanceTimeout,
			ReadBatchTimeout:      i.ReadBatchTimeout,
			MaxWait:               i.WaitBatchTimeout,
			RetentionTime:         i.GroupTTL,
			GroupBalancers:        []kafka.GroupBalancer{groupBalancer},
			Logger:                common.NewLogger(log),
			ErrorLogger:           common.NewErrorLogger(log),
		})

		i.readersPool[topic] = &topicReader{
			alias:         i.alias,
			pipe:          i.pipe,
			topic:         topic,
			groupId:       i.GroupId,
			clientId:      i.ClientId,
			enableMetrics: i.EnableMetrics,
			labelHeaders:  i.LabelHeaders,
			parser:        i.parser,
			ider:          i.Ider,

			commitSemaphore: semCh,
			reader:          reader,
			fetchCh:         fetchCh,
			commitCh:        commitCh,
			exitCh:          exitCh,
			doneCh:          doneCh,
			log:             log,
		}

		i.commitConsPool[topic] = &commitController{
			commitInterval: i.CommitInterval,
			commitQueue: orderedmap.New[int64, *trackedMessage](
				orderedmap.WithCapacity[int64, *trackedMessage](i.MaxUncommitted),
			),
			commitSemaphore: semCh,

			reader:   reader,
			fetchCh:  fetchCh,
			commitCh: commitCh,
			exitCh:   exitCh,
			doneCh:   doneCh,
			log:      log,
		}
	}

	i.fetchCtx, i.cancelFunc = context.WithCancel(context.Background())

	return nil
}

func (i *Kafka) SetChannels(out chan<- *core.Event) {
	for _, reader := range i.readersPool {
		reader.out = out
	}
}

func (i *Kafka) SetParser(p core.Parser) {
	i.parser = p
}

func (i *Kafka) Run() {
	for topic := range i.readersPool {
		i.wg.Add(1)
		go func(r *topicReader) {
			defer i.wg.Done()
			r.Run(i.fetchCtx)
		}(i.readersPool[topic])

		i.wg.Add(1)
		go func(c *commitController) {
			defer i.wg.Done()
			c.Run()
		}(i.commitConsPool[topic])
	}

	i.wg.Wait()
}

func (i *Kafka) Close() error {
	i.cancelFunc()
	return nil
}

func init() {
	plugins.AddInput("kafka", func() core.Input {
		return &Kafka{
			ClientId:          "neptunus.kafka.input",
			GroupId:           "neptunus.kafka.input",
			GroupBalancer:     "range",
			StartOffset:       "last",
			GroupTTL:          24 * time.Hour,
			DialTimeout:       5 * time.Second,
			SessionTimeout:    30 * time.Second,
			RebalanceTimeout:  30 * time.Second,
			HeartbeatInterval: 3 * time.Second,
			ReadBatchTimeout:  3 * time.Second,
			WaitBatchTimeout:  3 * time.Second,
			MaxUncommitted:    100,
			CommitInterval:    300 * time.Millisecond,
			MaxBatchSize:      1_048_576, // 1 MiB,
			SASL: SASL{
				Mechanism: "none",
			},
			Ider:            &ider.Ider{},
			TLSClientConfig: &tls.TLSClientConfig{},
		}
	})
}
