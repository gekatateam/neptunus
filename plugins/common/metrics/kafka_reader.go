package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
)

var (
	kafkaReaderMetricsRegister  = &sync.Once{}
	kafkaReaderMetricsCollector = &kafkaReaderCollector{
		statFuncs: make(map[readerDescriptor]func() ReaderStats),
		mu:        &sync.Mutex{},
	}

	kafkaReaderMessagesCount *prometheus.CounterVec
	kafkaReaderBytesCount    *prometheus.CounterVec
	kafkaReaderErrorsCount   *prometheus.CounterVec
	kafkaReaderTimeoutsCount *prometheus.CounterVec
	kafkaReaderFetchesCount  *prometheus.CounterVec

	kafkaReaderOffset              *prometheus.GaugeVec
	kafkaReaderLag                 *prometheus.GaugeVec
	kafkaReaderDelay               *prometheus.GaugeVec
	kafkaReaderCommitQueueCapacity *prometheus.GaugeVec
	kafkaReaderCommitQueueLength   *prometheus.GaugeVec

	kafkaReaderDialSecondsCount *prometheus.CounterVec
	kafkaReaderDialSecondsSum   *prometheus.CounterVec
	kafkaReaderDialSecondsMin   *prometheus.GaugeVec
	kafkaReaderDialSecondsAvg   *prometheus.GaugeVec
	kafkaReaderDialSecondsMax   *prometheus.GaugeVec

	kafkaReaderReadSecondsCount *prometheus.CounterVec
	kafkaReaderReadSecondsSum   *prometheus.CounterVec
	kafkaReaderReadSecondsMin   *prometheus.GaugeVec
	kafkaReaderReadSecondsAvg   *prometheus.GaugeVec
	kafkaReaderReadSecondsMax   *prometheus.GaugeVec

	kafkaReaderWaitSecondsCount *prometheus.CounterVec
	kafkaReaderWaitSecondsSum   *prometheus.CounterVec
	kafkaReaderWaitSecondsMin   *prometheus.GaugeVec
	kafkaReaderWaitSecondsAvg   *prometheus.GaugeVec
	kafkaReaderWaitSecondsMax   *prometheus.GaugeVec

	kafkaReaderFetchSizeCount *prometheus.CounterVec
	kafkaReaderFetchSizeSum   *prometheus.CounterVec
	kafkaReaderFetchSizeMin   *prometheus.GaugeVec
	kafkaReaderFetchSizeAvg   *prometheus.GaugeVec
	kafkaReaderFetchSizeMax   *prometheus.GaugeVec

	kafkaReaderFetchBytesCount *prometheus.CounterVec
	kafkaReaderFetchBytesSum   *prometheus.CounterVec
	kafkaReaderFetchBytesMin   *prometheus.GaugeVec
	kafkaReaderFetchBytesAvg   *prometheus.GaugeVec
	kafkaReaderFetchBytesMax   *prometheus.GaugeVec
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
	kafkaReaderDelay = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_delay",
			Help: "Reader current delay in milliseconds",
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

	kafkaReaderReadSecondsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_reader_read_seconds_count",
			Help: "Number of reads performed by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderReadSecondsSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_reader_read_seconds_sum",
			Help: "Total time spent on reads",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderReadSecondsMin = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_read_seconds_min",
			Help: "Min time spent on reads",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderReadSecondsAvg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_read_seconds_avg",
			Help: "Average time spent on reads",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderReadSecondsMax = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_read_seconds_max",
			Help: "Max time spent on reads",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)

	kafkaReaderWaitSecondsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_reader_wait_seconds_count",
			Help: "Number of message waiting cycles",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderWaitSecondsSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_reader_wait_seconds_sum",
			Help: "Total time spent on waiting for messages",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderWaitSecondsMin = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_wait_seconds_min",
			Help: "Min time spent on waiting",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderWaitSecondsAvg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_wait_seconds_avg",
			Help: "Average time spent on waiting",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderWaitSecondsMax = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_wait_seconds_max",
			Help: "Max time spent on waiting",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)

	kafkaReaderFetchSizeCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_reader_fetch_size_count",
			Help: "Number of fetches performed by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderFetchSizeSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_reader_fetch_size_sum",
			Help: "Total messages fetched by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderFetchSizeMin = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_fetch_size_min",
			Help: "Min messages fetched by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderFetchSizeAvg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_fetch_size_avg",
			Help: "Average messages fetched by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderFetchSizeMax = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_fetch_size_max",
			Help: "Max messages fetched by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)

	kafkaReaderFetchBytesCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_reader_fetch_bytes_count",
			Help: "Number of fetches performed by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderFetchBytesSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_reader_fetch_bytes_sum",
			Help: "Total bytes fetched by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderFetchBytesMin = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_fetch_bytes_min",
			Help: "Min bytes fetched by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderFetchBytesAvg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_fetch_bytes_avg",
			Help: "Average bytes fetched by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
	kafkaReaderFetchBytesMax = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_reader_fetch_bytes_max",
			Help: "Max bytes fetched by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "partition", "group_id", "client_id"},
	)
}

type ReaderStats struct {
	kafka.ReaderStats
	CommitQueueLenght   int
	CommitQueueCapacity int
	Delay               int64
}

type readerDescriptor struct {
	pipeline   string
	pluginName string
	topic      string
	partition  string
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
		kafkaReaderDelay.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Set(float64(stats.Delay))
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

		kafkaReaderDialSecondsCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.DialTime.Count))
		kafkaReaderDialSecondsSum.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(stats.DialTime.Sum.Seconds())
		kafkaReaderDialSecondsAvg.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(stats.DialTime.Avg.Seconds())
		kafkaReaderDialSecondsMin.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(stats.DialTime.Min.Seconds())
		kafkaReaderDialSecondsMax.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(stats.DialTime.Max.Seconds())

		kafkaReaderReadSecondsCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.ReadTime.Count))
		kafkaReaderReadSecondsSum.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(stats.ReadTime.Sum.Seconds())
		kafkaReaderReadSecondsAvg.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(stats.ReadTime.Avg.Seconds())
		kafkaReaderReadSecondsMin.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(stats.ReadTime.Min.Seconds())
		kafkaReaderReadSecondsMax.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(stats.ReadTime.Max.Seconds())

		kafkaReaderWaitSecondsCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.WaitTime.Count))
		kafkaReaderWaitSecondsSum.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(stats.WaitTime.Sum.Seconds())
		kafkaReaderWaitSecondsAvg.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(stats.WaitTime.Avg.Seconds())
		kafkaReaderWaitSecondsMin.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(stats.WaitTime.Min.Seconds())
		kafkaReaderWaitSecondsMax.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(stats.WaitTime.Max.Seconds())

		kafkaReaderFetchSizeCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.FetchSize.Count))
		kafkaReaderFetchSizeSum.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.FetchSize.Sum))
		kafkaReaderFetchSizeAvg.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.FetchSize.Avg))
		kafkaReaderFetchSizeMin.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.FetchSize.Min))
		kafkaReaderFetchSizeMax.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.FetchSize.Max))

		kafkaReaderFetchBytesCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.FetchBytes.Count))
		kafkaReaderFetchBytesSum.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.FetchBytes.Sum))
		kafkaReaderFetchBytesAvg.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.FetchBytes.Avg))
		kafkaReaderFetchBytesMin.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.FetchBytes.Min))
		kafkaReaderFetchBytesMax.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, stats.Partition,
			desc.groupId, desc.clientId,
		).Add(float64(stats.FetchBytes.Max))
	}
}

func RegisterKafkaReader(pipeline, pluginName, topic, partition, groupId, clientId string, statFunc func() ReaderStats) {
	kafkaReaderMetricsRegister.Do(func() {
		prometheus.MustRegister(kafkaReaderMessagesCount)
		prometheus.MustRegister(kafkaReaderBytesCount)
		prometheus.MustRegister(kafkaReaderErrorsCount)
		prometheus.MustRegister(kafkaReaderFetchesCount)
		prometheus.MustRegister(kafkaReaderTimeoutsCount)

		prometheus.MustRegister(kafkaReaderOffset)
		prometheus.MustRegister(kafkaReaderLag)
		prometheus.MustRegister(kafkaReaderDelay)
		prometheus.MustRegister(kafkaReaderCommitQueueCapacity)
		prometheus.MustRegister(kafkaReaderCommitQueueLength)

		prometheus.MustRegister(kafkaReaderDialSecondsCount)
		prometheus.MustRegister(kafkaReaderDialSecondsSum)
		prometheus.MustRegister(kafkaReaderDialSecondsMin)
		prometheus.MustRegister(kafkaReaderDialSecondsAvg)
		prometheus.MustRegister(kafkaReaderDialSecondsMax)

		prometheus.MustRegister(kafkaReaderReadSecondsCount)
		prometheus.MustRegister(kafkaReaderReadSecondsSum)
		prometheus.MustRegister(kafkaReaderReadSecondsMin)
		prometheus.MustRegister(kafkaReaderReadSecondsAvg)
		prometheus.MustRegister(kafkaReaderReadSecondsMax)

		prometheus.MustRegister(kafkaReaderWaitSecondsCount)
		prometheus.MustRegister(kafkaReaderWaitSecondsSum)
		prometheus.MustRegister(kafkaReaderWaitSecondsMin)
		prometheus.MustRegister(kafkaReaderWaitSecondsAvg)
		prometheus.MustRegister(kafkaReaderWaitSecondsMax)

		prometheus.MustRegister(kafkaReaderFetchSizeCount)
		prometheus.MustRegister(kafkaReaderFetchSizeSum)
		prometheus.MustRegister(kafkaReaderFetchSizeMin)
		prometheus.MustRegister(kafkaReaderFetchSizeAvg)
		prometheus.MustRegister(kafkaReaderFetchSizeMax)

		prometheus.MustRegister(kafkaReaderFetchBytesCount)
		prometheus.MustRegister(kafkaReaderFetchBytesSum)
		prometheus.MustRegister(kafkaReaderFetchBytesMin)
		prometheus.MustRegister(kafkaReaderFetchBytesAvg)
		prometheus.MustRegister(kafkaReaderFetchBytesMax)

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
		partition:  partition,
		groupId:    groupId,
		clientId:   clientId,
	}, statFunc)
}

func UnregisterKafkaReader(pipeline, pluginName, topic, partition, groupId, clientId string) {
	kafkaReaderMetricsCollector.delete(readerDescriptor{
		pipeline:   pipeline,
		pluginName: pluginName,
		topic:      topic,
		partition:  partition,
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
