package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
)

var (
	kafkaWriterMetricsRegister  = &sync.Once{}
	kafkaWriterMetricsCollector = &kafkaWriterCollector{
		statFuncs: make(map[writerDescriptor]func() kafka.WriterStats),
	}

	kafkaWriterWritesCount   *prometheus.Desc // counter
	kafkaWriterMessagesCount *prometheus.Desc
	kafkaWriterBytesCount    *prometheus.Desc
	kafkaWriterErrorsCount   *prometheus.Desc

	kafkaWriterBatchSecondsCount *prometheus.Desc // counter
	kafkaWriterBatchSecondsSum   *prometheus.Desc
	kafkaWriterBatchSecondsMin   *prometheus.Desc // gauge
	kafkaWriterBatchSecondsAvg   *prometheus.Desc
	kafkaWriterBatchSecondsMax   *prometheus.Desc

	kafkaWriterBatchQueueSecondsCount *prometheus.Desc // counter
	kafkaWriterBatchQueueSecondsSum   *prometheus.Desc
	kafkaWriterBatchQueueSecondsMin   *prometheus.Desc // gauge
	kafkaWriterBatchQueueSecondsAvg   *prometheus.Desc
	kafkaWriterBatchQueueSecondsMax   *prometheus.Desc

	kafkaWriterBatchSizeMin *prometheus.Desc // gauge
	kafkaWriterBatchSizeAvg *prometheus.Desc
	kafkaWriterBatchSizeMax *prometheus.Desc

	kafkaWriterBatchBytesMin *prometheus.Desc // gauge
	kafkaWriterBatchBytesAvg *prometheus.Desc
	kafkaWriterBatchBytesMax *prometheus.Desc
)

func init() {
	kafkaWriterWritesCount = prometheus.NewDesc(
		"plugin_kafka_writer_writes_count",
		"Number of writes performed by client",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterMessagesCount = prometheus.NewDesc(
		"plugin_kafka_writer_messages_count",
		"Number of messages written by client",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterBytesCount = prometheus.NewDesc(
		"plugin_kafka_writer_bytes_count",
		"Number of bytes written by client",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterErrorsCount = prometheus.NewDesc(
		"plugin_kafka_writer_errors_count",
		"Number of errors occurred during writing",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)

	kafkaWriterBatchSecondsCount = prometheus.NewDesc(
		"plugin_kafka_writer_batch_seconds_count",
		"Number of batches written by client",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterBatchSecondsSum = prometheus.NewDesc(
		"plugin_kafka_writer_batch_seconds_sum",
		"Total time spent on batches write",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterBatchSecondsMin = prometheus.NewDesc(
		"plugin_kafka_writer_batch_seconds_min",
		"Min time spent on batches write",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterBatchSecondsAvg = prometheus.NewDesc(
		"plugin_kafka_writer_batch_seconds_avg",
		"Average time spent on batches write",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterBatchSecondsMax = prometheus.NewDesc(
		"plugin_kafka_writer_batch_seconds_max",
		"Max time spent on batches write",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)

	kafkaWriterBatchQueueSecondsCount = prometheus.NewDesc(
		"plugin_kafka_writer_batch_queue_seconds_count",
		"Number of queued batches",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterBatchQueueSecondsSum = prometheus.NewDesc(
		"plugin_kafka_writer_batch_queue_seconds_sum",
		"Total time spent in queue",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterBatchQueueSecondsMin = prometheus.NewDesc(
		"plugin_kafka_writer_batch_queue_seconds_min",
		"Min time spent in queue",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterBatchQueueSecondsAvg = prometheus.NewDesc(
		"plugin_kafka_writer_batch_queue_seconds_avg",
		"Average time spent in queue",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterBatchQueueSecondsMax = prometheus.NewDesc(
		"plugin_kafka_writer_batch_queue_seconds_max",
		"Max time spent in queue",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)

	kafkaWriterBatchSizeMin = prometheus.NewDesc(
		"plugin_kafka_writer_batch_size_min",
		"Min written batch size",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterBatchSizeAvg = prometheus.NewDesc(
		"plugin_kafka_writer_batch_size_avg",
		"Average written batch size",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterBatchSizeMax = prometheus.NewDesc(
		"plugin_kafka_writer_batch_size_max",
		"Max written batch size",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)

	kafkaWriterBatchBytesMin = prometheus.NewDesc(
		"plugin_kafka_writer_batch_bytes_min",
		"Min written batch bytes",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterBatchBytesAvg = prometheus.NewDesc(
		"plugin_kafka_writer_batch_bytes_avg",
		"Average written batch bytes",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterBatchBytesMax = prometheus.NewDesc(
		"plugin_kafka_writer_batch_bytes_max",
		"Max written batch bytes",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
}

type writerDescriptor struct {
	pipeline   string
	pluginName string
	clientId   string
}

type kafkaWriterCollector struct {
	statFuncs map[writerDescriptor]func() kafka.WriterStats
}

func (c *kafkaWriterCollector) Collect(ch chan<- prometheus.Metric) {}
func (c *kafkaWriterCollector) Describe(ch chan<- *prometheus.Desc) {}

func (c *kafkaWriterCollector) append(d writerDescriptor, f func() kafka.WriterStats) {
	c.statFuncs[d] = f
}

func (c *kafkaWriterCollector) delete(d writerDescriptor) {
	delete(c.statFuncs, d)
}

func RegisterKafkaWriter(pipeline, pluginName, clientId string, statFunc func() kafka.WriterStats) {
	kafkaWriterMetricsRegister.Do(func() {
		prometheus.MustRegister(kafkaWriterMetricsCollector)
	})

	kafkaWriterMetricsCollector.append(writerDescriptor{
		pipeline:   pipeline,
		pluginName: pluginName,
		clientId:   clientId,
	}, statFunc)
}

func UnregisterKafkaWriter(pipeline, pluginName, clientId string) {
	kafkaWriterMetricsCollector.delete(writerDescriptor{
		pipeline:   pipeline,
		pluginName: pluginName,
		clientId:   clientId,
	})
}

// type WriterStats struct {
//     Writes   int64 `metric:"kafka.writer.write.count"     type:"counter"`
//     Messages int64 `metric:"kafka.writer.message.count"   type:"counter"`
//     Bytes    int64 `metric:"kafka.writer.message.bytes"   type:"counter"`
//     Errors   int64 `metric:"kafka.writer.error.count"     type:"counter"`

//     BatchTime      DurationStats `metric:"kafka.writer.batch.seconds"`
//     BatchQueueTime DurationStats `metric:"kafka.writer.batch.queue.seconds"`
//     WriteTime      DurationStats `metric:"kafka.writer.write.seconds"`
//     WaitTime       DurationStats `metric:"kafka.writer.wait.seconds"`
//     Retries        int64         `metric:"kafka.writer.retries.count" type:"counter"`
//     BatchSize      SummaryStats  `metric:"kafka.writer.batch.size"`
//     BatchBytes     SummaryStats  `metric:"kafka.writer.batch.bytes"`

//     MaxAttempts     int64         `metric:"kafka.writer.attempts.max"  type:"gauge"`
//     WriteBackoffMin time.Duration `metric:"kafka.writer.backoff.min"   type:"gauge"`
//     WriteBackoffMax time.Duration `metric:"kafka.writer.backoff.max"   type:"gauge"`
//     MaxBatchSize    int64         `metric:"kafka.writer.batch.max"     type:"gauge"`
//     BatchTimeout    time.Duration `metric:"kafka.writer.batch.timeout" type:"gauge"`
//     ReadTimeout     time.Duration `metric:"kafka.writer.read.timeout"  type:"gauge"`
//     WriteTimeout    time.Duration `metric:"kafka.writer.write.timeout" type:"gauge"`
//     RequiredAcks    int64         `metric:"kafka.writer.acks.required" type:"gauge"`
//     Async           bool          `metric:"kafka.writer.async"         type:"gauge"`

//     Topic string `tag:"topic"`

//     // DEPRECATED: these fields will only be reported for backward compatibility
//     // if the Writer was constructed with NewWriter.
//     Dials    int64         `metric:"kafka.writer.dial.count" type:"counter"`
//     DialTime DurationStats `metric:"kafka.writer.dial.seconds"`

//     // DEPRECATED: these fields were meaningful prior to kafka-go 0.4, changes
//     // to the internal implementation and the introduction of the transport type
//     // made them unnecessary.
//     //
//     // The values will be zero but are left for backward compatibility to avoid
//     // breaking programs that used these fields.
//     Rebalances        int64
//     RebalanceInterval time.Duration
//     QueueLength       int64
//     QueueCapacity     int64
//     ClientID          string
// }
