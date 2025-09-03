package kafka

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"kythe.io/kythe/go/util/datasize"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/ider"
	common "github.com/gekatateam/neptunus/plugins/common/kafka"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

type Kafka struct {
	*core.BaseInput      `mapstructure:"-"`
	EnableMetrics        bool              `mapstructure:"enable_metrics"`
	Brokers              []string          `mapstructure:"brokers"`
	OnParserError        string            `mapstructure:"on_parser_error"`
	ClientId             string            `mapstructure:"client_id"`
	GroupId              string            `mapstructure:"group_id"`
	GroupTTL             time.Duration     `mapstructure:"group_ttl"`
	GroupBalancer        string            `mapstructure:"group_balancer"`
	Rack                 string            `mapstructure:"rack"`
	Topics               []string          `mapstructure:"topics"`
	KeepTimestamp        bool              `mapstructure:"keep_timestamp"`
	DialTimeout          time.Duration     `mapstructure:"dial_timeout"`
	SessionTimeout       time.Duration     `mapstructure:"session_timeout"`
	RebalanceTimeout     time.Duration     `mapstructure:"rebalance_timeout"`
	HeartbeatInterval    time.Duration     `mapstructure:"heartbeat_interval"`
	ReadBatchTimeout     time.Duration     `mapstructure:"read_batch_timeout"`
	WaitBatchTimeout     time.Duration     `mapstructure:"wait_batch_timeout"`
	StartOffset          string            `mapstructure:"start_offset"`
	MaxBatchSize         datasize.Size     `mapstructure:"max_batch_size"`
	MaxUncommitted       int               `mapstructure:"max_uncommitted"`
	CommitInterval       time.Duration     `mapstructure:"commit_interval"`
	SASL                 SASL              `mapstructure:"sasl"`
	LabelHeaders         map[string]string `mapstructure:"labelheaders"`
	*ider.Ider           `mapstructure:",squash"`
	*tls.TLSClientConfig `mapstructure:",squash"`

	dialer   *kafka.Dialer
	cgConfig kafka.ConsumerGroupConfig

	fetchCtx   context.Context
	cancelFunc context.CancelFunc

	parser core.Parser
}

type SASL struct {
	Mechanism string `mapstructure:"mechanism"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
}

func (i *Kafka) Init() (err error) {
	if len(i.Brokers) == 0 {
		return errors.New("at least one broker address required")
	}

	if len(i.Topics) == 0 {
		return errors.New("at least one topic required")
	}

	if len(i.GroupId) == 0 {
		return errors.New("group_id required")
	}

	switch i.OnParserError {
	case "drop":
	case "consume":
	default:
		return fmt.Errorf("unknown parser error behaviour: %v; expected one of: drop, consume", i.OnParserError)
	}

	if err := i.Ider.Init(); err != nil {
		return err
	}

	if i.MaxUncommitted < 0 {
		i.MaxUncommitted = 100
	}

	i.Topics = slices.Compact(i.Topics)

	var m sasl.Mechanism
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

	i.dialer = &kafka.Dialer{
		ClientID:      i.ClientId,
		DualStack:     true,
		Timeout:       i.DialTimeout,
		SASLMechanism: m,
		TLS:           tlsConfig,
	}

	if err := i.testConn(i.dialer); err != nil {
		return err
	}

	i.cgConfig = kafka.ConsumerGroupConfig{
		ID:                    i.GroupId,
		Brokers:               i.Brokers,
		Topics:                i.Topics,
		Dialer:                i.dialer,
		WatchPartitionChanges: true,
		StartOffset:           offset,
		HeartbeatInterval:     i.HeartbeatInterval,
		SessionTimeout:        i.SessionTimeout,
		RebalanceTimeout:      i.RebalanceTimeout,
		RetentionTime:         i.GroupTTL,
		GroupBalancers:        []kafka.GroupBalancer{groupBalancer},
		Logger:                common.NewLogger(i.Log),
		ErrorLogger:           common.NewErrorLogger(i.Log),
	}

	if err := i.cgConfig.Validate(); err != nil {
		return err
	}

	i.fetchCtx, i.cancelFunc = context.WithCancel(context.Background())

	return nil
}

func (i *Kafka) SetParser(p core.Parser) {
	i.parser = p
}

func (i *Kafka) Run() {
	wg := &sync.WaitGroup{}
	cg, _ := kafka.NewConsumerGroup(i.cgConfig) // because config validated before, no err possible here
	defer cg.Close()

GENERATION_LOOP:
	for {
		gen, err := cg.Next(i.fetchCtx)
		switch {
		case err == nil:
			i.Log.Info(fmt.Sprintf("new generation started with member id: %v, generation id: %v", gen.MemberID, gen.ID))
		case errors.Is(err, context.Canceled):
			i.Log.Info("stop signal received, breaking generation loop")
			cg.Close()
			break GENERATION_LOOP
		default:
			i.Log.Warn("consumer group closed, maybe because Kafka cluster is shutting down or network problems",
				"error", err,
			)
			time.Sleep(i.DialTimeout)
			continue GENERATION_LOOP
		}

		for topic, topicAssigment := range gen.Assignments {
			var (
				fetchCh  = make(chan *trackedMessage)
				commitCh = make(chan commitMessage)
				exitCh   = make(chan struct{})
				doneCh   = make(chan struct{})
				semCh    = make(chan struct{}, i.MaxUncommitted)
			)

			wg.Add(1)
			gen.Start(func(_ context.Context) {
				defer wg.Done()
				commiter := &commitController{
					commitInterval:  i.CommitInterval,
					commitQueues:    make(map[int]*orderedmap.OrderedMap[int64, *trackedMessage]),
					commitSemaphore: semCh,

					gen:      gen,
					fetchCh:  fetchCh,
					commitCh: commitCh,
					exitCh:   exitCh,
					doneCh:   doneCh,
					log:      i.Log,
				}

				commiter.Run()
			})

			for _, partitionAssigment := range topicAssigment {
				wg.Add(1)
				gen.Start(func(ctx context.Context) {
					defer wg.Done()
					reader := &topicReader{
						BaseInput:     i.BaseInput,
						topic:         topic,
						partition:     strconv.Itoa(partitionAssigment.ID),
						groupId:       i.GroupId,
						clientId:      i.ClientId,
						enableMetrics: i.EnableMetrics,
						labelHeaders:  i.LabelHeaders,
						keepTimestamp: i.KeepTimestamp,
						onParserError: i.OnParserError,
						parser:        i.parser,
						ider:          i.Ider,

						commitSemaphore: semCh,
						fetchCh:         fetchCh,
						commitCh:        commitCh,
						exitCh:          exitCh,
						doneCh:          doneCh,

						reader: kafka.NewReader(kafka.ReaderConfig{
							Brokers:          i.Brokers,
							Topic:            topic,
							Partition:        partitionAssigment.ID,
							MaxBytes:         int(i.MaxBatchSize.Bytes()),
							QueueCapacity:    100, // <-
							MaxAttempts:      1,
							Dialer:           i.dialer,
							ReadBatchTimeout: i.ReadBatchTimeout,
							MaxWait:          i.WaitBatchTimeout,
							Logger:           common.NewLogger(i.Log),
							ErrorLogger:      common.NewErrorLogger(i.Log),
						}),
					}
					reader.reader.SetOffset(partitionAssigment.Offset)
					reader.Run(ctx)
				})
			}
		}
	}

	wg.Wait()
}

func (i *Kafka) Close() error {
	i.cancelFunc()
	return nil
}

func (i *Kafka) testConn(dialer *kafka.Dialer) error {
	ctx, cancel := context.WithTimeout(context.Background(), i.DialTimeout)
	defer cancel()

	conn, err := dialer.DialContext(ctx, "tcp", i.Brokers[0])
	if err != nil {
		return err
	}

	conn.Close()
	return nil
}

func init() {
	plugins.AddInput("kafka", func() core.Input {
		return &Kafka{
			ClientId:          "neptunus.kafka.input",
			GroupId:           "neptunus.kafka.input",
			GroupBalancer:     "range",
			StartOffset:       "last",
			OnParserError:     "drop",
			GroupTTL:          24 * time.Hour,
			DialTimeout:       5 * time.Second,
			SessionTimeout:    30 * time.Second,
			RebalanceTimeout:  30 * time.Second,
			HeartbeatInterval: 3 * time.Second,
			ReadBatchTimeout:  3 * time.Second,
			WaitBatchTimeout:  3 * time.Second,
			MaxUncommitted:    100,
			CommitInterval:    1 * time.Second,
			MaxBatchSize:      datasize.Mebibyte, // 1 MiB,
			SASL: SASL{
				Mechanism: "none",
			},
			Ider:            &ider.Ider{},
			TLSClientConfig: &tls.TLSClientConfig{},
		}
	})
}
