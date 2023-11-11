package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
)

type Kafka struct {
	alias             string
	pipe              string
	EnableMetrics     bool              `mapstructure:"enable_metrics"`
	Brokers           []string          `mapstructure:"brokers"`
	ClientId          string            `mapstructure:"client_id"`
	GroupId           string            `mapstructure:"group_id"`
	GroupTTL          time.Duration     `mapstructure:"group_ttl"`
	GroupBalancer     string            `mapstructure:"group_balancer"`
	Rack              string            `mapstructure:"rack"`
	Topics            []string          `mapstructure:"topics"`
	DialTimeout       time.Duration     `mapstructure:"dial_timeout"`
	SessionTimeout    time.Duration     `mapstructure:"session_timeout"`
	RebalanceTimeout  time.Duration     `mapstructure:"rebalance_timeout"`
	HeartbeatInterval time.Duration     `mapstructure:"heartbeat_interval"`
	StartOffset       string            `mapstructure:"start_offset"`
	MaxBatchSize      int               `mapstructure:"max_batch_size"`
	MaxUncommitted    int               `mapstructure:"max_uncommitted"`
	SASL              SASL              `mapstructure:"sasl"`
	LabelHeaders      map[string]string `mapstructure:"labelheaders"`

	readersPool map[string]*topicReader
	fetchCtx    context.Context
	cancelFunc  context.CancelFunc
	wg          *sync.WaitGroup

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

	var offset int64
	switch i.StartOffset {
	case "first":
		offset = kafka.FirstOffset
	case "last":
		offset = kafka.LastOffset
	default:
		return fmt.Errorf("unknown offset: %v; expected one of: first, last", i.StartOffset)
	}

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

		dialer := &kafka.Dialer{
			ClientID:      i.ClientId,
			DualStack:     true,
			Timeout:       i.DialTimeout,
			SASLMechanism: m,
		}

		readerConfig := kafka.ReaderConfig{
			Brokers:               i.Brokers,
			GroupID:               i.GroupId,
			Topic:                 topic,
			Dialer:                dialer,
			QueueCapacity:         1,
			MaxAttempts:           1,
			WatchPartitionChanges: true,
			StartOffset:           offset,
			HeartbeatInterval:     i.HeartbeatInterval,
			SessionTimeout:        i.SessionTimeout,
			RebalanceTimeout:      i.RebalanceTimeout,
			RetentionTime:         i.GroupTTL,
			GroupBalancers:        []kafka.GroupBalancer{groupBalancer},
		}

		i.readersPool[topic] = &topicReader{
			alias:         i.alias,
			pipe:          i.pipe,
			topic:         topic,
			groupId:       i.GroupId,
			clientId:      i.ClientId,
			enableMetrics: i.EnableMetrics,
			labelHeaders:  i.LabelHeaders,
			reader:        kafka.NewReader(readerConfig),
			sem: &commitSemaphore{
				ch: make(chan struct{}, i.MaxUncommitted),
				wg: &sync.WaitGroup{},
			},
			cQueue: &orderedmap.OrderedMap[int64, kafka.Message]{},
			log:    log,
		}
	}

	i.fetchCtx, i.cancelFunc = context.WithCancel(context.Background())

	return nil
}

func (i *Kafka) Prepare(out chan<- *core.Event) {
	for _, reader := range i.readersPool {
		reader.out = out
	}
}

func (i *Kafka) SetParser(p core.Parser) {
	for _, reader := range i.readersPool {
		reader.parser = p
	}
}

func (i *Kafka) Run() {
	for _, reader := range i.readersPool {
		go func(r *topicReader) {
			i.wg.Add(1)
			defer i.wg.Done()
			r.Run(i.fetchCtx)
		}(reader)
	}
}

func (i *Kafka) Close() error {
	i.cancelFunc()
	i.wg.Wait()
	return nil
}

func (i *Kafka) Alias() string {
	return i.alias
}

func init() {
	plugins.AddInput("kafka", func() core.Input {
		return &Kafka{
			readersPool: make(map[string]*topicReader),
			wg:          &sync.WaitGroup{},
		}
	})
}
