package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
)

// value of 'partition' lablel is always '-1' beacuse of https://github.com/segmentio/kafka-go/blob/v0.4.43/reader.go#L685s
var (
	kafkaReaderMetricsRegister  = &sync.Once{}
	kafkaReaderMetricsCollector = &kafkaReaderCollector{
		statFuncs: make(map[readerDescriptor]func() ReaderStats),
		mu:        &sync.Mutex{},
	}

	kafkaReaderMessagesCount   *prometheus.CounterVec
	kafkaReaderBytesCount      *prometheus.CounterVec
	kafkaReaderErrorsCount     *prometheus.CounterVec
	kafkaReaderRebalancesCount *prometheus.CounterVec
	kafkaReaderTimeoutsCount   *prometheus.CounterVec
	kafkaReaderFetchesCount    *prometheus.CounterVec

	kafkaReaderOffset              *prometheus.GaugeVec
	kafkaReaderLag                 *prometheus.GaugeVec
	kafkaReaderCommitQueueCapacity *prometheus.GaugeVec
	kafkaReaderCommitQueueLength   *prometheus.GaugeVec

	kafkaReaderDialSecondsCount *prometheus.CounterVec
	kafkaReaderDialSecondsSum   *prometheus.CounterVec
	kafkaReaderDialSecondsMin   *prometheus.GaugeVec
	kafkaReaderDialSecondsAvg   *prometheus.GaugeVec
	kafkaReaderDialSecondsMax   *prometheus.GaugeVec
)

func init() {
	kafkaReaderMessagesCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_reader_messages_count",
			Help: "Number of messages read by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderBytesCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_reader_bytes_count",
			Help: "Number of bytes read by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderErrorsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_reader_errors_count",
			Help: "Number of errors occurred during reads",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderRebalancesCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_reader_rebalances_count",
			Help: "Number of consumer rebalances",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderTimeoutsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_reader_timeouts_count",
			Help: "Number of fetches that ends with timeout",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderFetchesCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_reader_fetches_count",
			Help: "Total number of fetches done by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)

	kafkaReaderOffset = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_offset",
			Help: "Reader current offset",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderLag = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_lag",
			Help: "Reader current lag",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderCommitQueueCapacity = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_commit_queue_capacity",
			Help: "Reader internal commit queue capacity",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderCommitQueueLength = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_commit_queue_length",
			Help: "Reader internal commit queue length",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)

	kafkaReaderDialSecondsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_reader_dial_seconds_count",
			Help: "Number of dials performed by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderDialSecondsSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_reader_dial_seconds_sum",
			Help: "Total time spent on dials",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderDialSecondsMin = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_dial_seconds_min",
			Help: "Min time spent on dials",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderDialSecondsAvg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_dial_seconds_avg",
			Help: "Average time spent on dials",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderDialSecondsMax = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_dial_seconds_max",
			Help: "Max time spent on dials",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
}

type ReaderStats struct {
	kafka.ReaderStats
	CommitQueueLenght   int
	CommitQueueCapacity int
}

type readerDescriptor struct {
	pipeline   string
	pluginName string
	topic      string
	clientId   string
	groupId    string
}

type kafkaReaderCollector struct {
	statFuncs map[readerDescriptor]func() ReaderStats
	mu        *sync.Mutex
}

func (c *kafkaReaderCollector) append(d readerDescriptor, f func() ReaderStats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.statFuncs[d] = f
}

func (c *kafkaReaderCollector) delete(d readerDescriptor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.statFuncs, d)
}

func (c *kafkaReaderCollector) collect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for desc, f := range c.statFuncs {
		stats := f()

		kafkaReaderMessagesCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.Messages))
		kafkaReaderBytesCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.Bytes))
		kafkaReaderErrorsCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.Errors))
		kafkaReaderRebalancesCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.Rebalances))
		kafkaReaderFetchesCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.Fetches))
		kafkaReaderTimeoutsCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.Timeouts))

		kafkaReaderOffset.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Set(float64(stats.Offset))
		kafkaReaderLag.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Set(float64(stats.Lag))
		kafkaReaderCommitQueueCapacity.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Set(float64(stats.CommitQueueCapacity))
		kafkaReaderCommitQueueLength.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Set(float64(stats.CommitQueueLenght))
	}
}

func RegisterKafkaReader(pipeline, pluginName, topic, groupId, clientId string, statFunc func() ReaderStats) {
	kafkaReaderMetricsRegister.Do(func() {
		prometheus.MustRegister(kafkaReaderMessagesCount)
		prometheus.MustRegister(kafkaReaderBytesCount)
		prometheus.MustRegister(kafkaReaderErrorsCount)
		prometheus.MustRegister(kafkaReaderRebalancesCount)
		prometheus.MustRegister(kafkaReaderFetchesCount)
		prometheus.MustRegister(kafkaReaderTimeoutsCount)

		t := time.NewTicker(15 * time.Second)
		go func() {
			for range t.C {
				kafkaReaderMetricsCollector.collect()
			}
		}()
	})

	kafkaReaderMetricsCollector.append(readerDescriptor{
		pipeline:   pipeline,
		pluginName: pluginName,
		topic:      topic,
		groupId:    groupId,
		clientId:   clientId,
	}, statFunc)
}

func UnregisterKafkaReader(pipeline, pluginName, topic, groupId, clientId string) {
	kafkaReaderMetricsCollector.delete(readerDescriptor{
		pipeline:   pipeline,
		pluginName: pluginName,
		topic:      topic,
		groupId:    groupId,
		clientId:   clientId,
	})
}

// type ReaderStats struct {
//     Dials      int64 `metric:"kafka.reader.dial.count"      type:"counter"`
//     Fetches    int64 `metric:"kafka.reader.fetch.count"     type:"counter"`
//     Messages   int64 `metric:"kafka.reader.message.count"   type:"counter"`
//     Bytes      int64 `metric:"kafka.reader.message.bytes"   type:"counter"`
//     Rebalances int64 `metric:"kafka.reader.rebalance.count" type:"counter"`
//     Timeouts   int64 `metric:"kafka.reader.timeout.count"   type:"counter"`
//     Errors     int64 `metric:"kafka.reader.error.count"     type:"counter"`

//     DialTime   DurationStats `metric:"kafka.reader.dial.seconds"`
//     ReadTime   DurationStats `metric:"kafka.reader.read.seconds"`
//     WaitTime   DurationStats `metric:"kafka.reader.wait.seconds"`
//     FetchSize  SummaryStats  `metric:"kafka.reader.fetch.size"`
//     FetchBytes SummaryStats  `metric:"kafka.reader.fetch.bytes"`

//     Offset        int64         `metric:"kafka.reader.offset"          type:"gauge"`
//     Lag           int64         `metric:"kafka.reader.lag"             type:"gauge"`
//     MinBytes      int64         `metric:"kafka.reader.fetch_bytes.min" type:"gauge"`
//     MaxBytes      int64         `metric:"kafka.reader.fetch_bytes.max" type:"gauge"`
//     MaxWait       time.Duration `metric:"kafka.reader.fetch_wait.max"  type:"gauge"`
//     QueueLength   int64         `metric:"kafka.reader.queue.length"    type:"gauge"`
//     QueueCapacity int64         `metric:"kafka.reader.queue.capacity"  type:"gauge"`

//     ClientID  string `tag:"client_id"`
//     Topic     string `tag:"topic"`
//     Partition string `tag:"partition"`

//     // The original `Fetches` field had a typo where the metric name was called
//     // "kafak..." instead of "kafka...", in order to offer time to fix monitors
//     // that may be relying on this mistake we are temporarily introducing this
//     // field.
//     DeprecatedFetchesWithTypo int64 `metric:"kafak.reader.fetch.count" type:"counter"`
// }
