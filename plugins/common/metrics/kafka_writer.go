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

	kafkaWriterMessagesCount *prometheus.CounterVec
	kafkaWriterBytesCount    *prometheus.CounterVec
	kafkaWriterErrorsCount   *prometheus.CounterVec

	kafkaWriterWriteSecondsCount *prometheus.CounterVec // counter
	kafkaWriterWriteSecondsSum   *prometheus.CounterVec
	kafkaWriterWriteSecondsMin   *prometheus.Desc // gauge
	kafkaWriterWriteSecondsAvg   *prometheus.Desc
	kafkaWriterWriteSecondsMax   *prometheus.Desc

	// TODO what it means?
	kafkaWriterBatchSecondsCount *prometheus.CounterVec // counter
	kafkaWriterBatchSecondsSum   *prometheus.CounterVec
	kafkaWriterBatchSecondsMin   *prometheus.Desc // gauge
	kafkaWriterBatchSecondsAvg   *prometheus.Desc
	kafkaWriterBatchSecondsMax   *prometheus.Desc

	// TODO what it means?
	kafkaWriterBatchQueueSecondsCount *prometheus.CounterVec // counter
	kafkaWriterBatchQueueSecondsSum   *prometheus.CounterVec
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
	kafkaWriterMessagesCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_messages_count",
			Help: "Number of messages written by client",
		},
		[]string{"pipeline", "plugin_name", "client_id"},
	)
	kafkaWriterBytesCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_bytes_count",
			Help: "Number of bytes written by client",
		},
		[]string{"pipeline", "plugin_name", "client_id"},
	)
	kafkaWriterErrorsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_errors_count",
			Help: "Number of errors occurred during writing",
		},
		[]string{"pipeline", "plugin_name", "client_id"},
	)

	kafkaWriterBatchSecondsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_batch_seconds_count",
			Help: "Number of batches created by client",
		},
		[]string{"pipeline", "plugin_name", "client_id"},
	)
	kafkaWriterBatchSecondsSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_batch_seconds_sum",
			Help: "Total time spent on filling batches",
		},
		[]string{"pipeline", "plugin_name", "client_id"},
	)
	kafkaWriterBatchSecondsMin = prometheus.NewDesc(
		"plugin_kafka_writer_batch_seconds_min",
		"Min time spent on filling batches",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterBatchSecondsAvg = prometheus.NewDesc(
		"plugin_kafka_writer_batch_seconds_avg",
		"Average time spent on filling batches",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterBatchSecondsMax = prometheus.NewDesc(
		"plugin_kafka_writer_batch_seconds_max",
		"Max time spent on filling batches",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)

	kafkaWriterWriteSecondsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_write_seconds_count",
			Help: "Number of writes performed by client",
		},
		[]string{"pipeline", "plugin_name", "client_id"},
	)
	kafkaWriterWriteSecondsSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_write_seconds_sum",
			Help: "Total time spent on writes",
		},
		[]string{"pipeline", "plugin_name", "client_id"},
	)
	kafkaWriterWriteSecondsMin = prometheus.NewDesc(
		"plugin_kafka_writer_write_seconds_min",
		"Min time spent on writes",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterWriteSecondsAvg = prometheus.NewDesc(
		"plugin_kafka_writer_write_seconds_avg",
		"Average time spent on writes",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)
	kafkaWriterWriteSecondsMax = prometheus.NewDesc(
		"plugin_kafka_writer_write_seconds_max",
		"Max time spent on writes",
		[]string{"pipeline", "plugin_name", "client_id"},
		nil,
	)

	kafkaWriterBatchQueueSecondsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_batch_queue_seconds_count",
			Help: "Number of batches placed in queue",
		},
		[]string{"pipeline", "plugin_name", "client_id"},
	)
	kafkaWriterBatchQueueSecondsSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_batch_queue_seconds_sum",
			Help: "Total time spent in queue",
		},
		[]string{"pipeline", "plugin_name", "client_id"},
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

func (c *kafkaWriterCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- kafkaWriterBatchSecondsMin
	ch <- kafkaWriterBatchSecondsAvg
	ch <- kafkaWriterBatchSecondsMax

	ch <- kafkaWriterWriteSecondsMin
	ch <- kafkaWriterWriteSecondsAvg
	ch <- kafkaWriterWriteSecondsMax

	ch <- kafkaWriterBatchQueueSecondsMin
	ch <- kafkaWriterBatchQueueSecondsAvg
	ch <- kafkaWriterBatchQueueSecondsMax

	ch <- kafkaWriterBatchSizeMin
	ch <- kafkaWriterBatchSizeAvg
	ch <- kafkaWriterBatchSizeMax

	ch <- kafkaWriterBatchBytesMin
	ch <- kafkaWriterBatchBytesAvg
	ch <- kafkaWriterBatchBytesMax
}

func (c *kafkaWriterCollector) Collect(ch chan<- prometheus.Metric) {
	for desc, f := range c.statFuncs {
		stats := f()

		println("TOPIC: "+stats.Topic)

		// it is a hack for keeping counter totals
		kafkaWriterMessagesCount.WithLabelValues(
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		).Add(float64(stats.Messages))
		kafkaWriterBytesCount.WithLabelValues(
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		).Add(float64(stats.Bytes))
		kafkaWriterErrorsCount.WithLabelValues(
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		).Add(float64(stats.Errors))

		kafkaWriterBatchSecondsCount.WithLabelValues(
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		).Add(float64(stats.BatchTime.Count))
		kafkaWriterBatchSecondsSum.WithLabelValues(
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		).Add(stats.BatchTime.Sum.Seconds())
		ch <- prometheus.MustNewConstMetric(
			kafkaWriterBatchSecondsMin,
			prometheus.GaugeValue,
			stats.BatchTime.Min.Seconds(),
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		)
		ch <- prometheus.MustNewConstMetric(
			kafkaWriterBatchSecondsAvg,
			prometheus.GaugeValue,
			stats.BatchTime.Avg.Seconds(),
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		)
		ch <- prometheus.MustNewConstMetric(
			kafkaWriterBatchSecondsMax,
			prometheus.GaugeValue,
			stats.BatchTime.Max.Seconds(),
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		)

		kafkaWriterWriteSecondsCount.WithLabelValues(
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		).Add(float64(stats.WriteTime.Count))
		kafkaWriterWriteSecondsSum.WithLabelValues(
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		).Add(stats.WriteTime.Sum.Seconds())
		ch <- prometheus.MustNewConstMetric(
			kafkaWriterWriteSecondsMin,
			prometheus.GaugeValue,
			stats.WriteTime.Min.Seconds(),
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		)
		ch <- prometheus.MustNewConstMetric(
			kafkaWriterWriteSecondsAvg,
			prometheus.GaugeValue,
			stats.WriteTime.Avg.Seconds(),
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		)
		ch <- prometheus.MustNewConstMetric(
			kafkaWriterWriteSecondsMax,
			prometheus.GaugeValue,
			stats.WriteTime.Max.Seconds(),
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		)

		kafkaWriterBatchQueueSecondsCount.WithLabelValues(
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		).Add(float64(stats.BatchQueueTime.Count))
		kafkaWriterBatchQueueSecondsSum.WithLabelValues(
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		).Add(stats.BatchQueueTime.Sum.Seconds())
		ch <- prometheus.MustNewConstMetric(
			kafkaWriterBatchQueueSecondsMin,
			prometheus.GaugeValue,
			stats.BatchQueueTime.Min.Seconds(),
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		)
		ch <- prometheus.MustNewConstMetric(
			kafkaWriterBatchQueueSecondsAvg,
			prometheus.GaugeValue,
			stats.BatchQueueTime.Avg.Seconds(),
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		)
		ch <- prometheus.MustNewConstMetric(
			kafkaWriterBatchQueueSecondsMax,
			prometheus.GaugeValue,
			stats.BatchQueueTime.Max.Seconds(),
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		)
	
		ch <- prometheus.MustNewConstMetric(
			kafkaWriterBatchSizeMin,
			prometheus.GaugeValue,
			float64(stats.BatchSize.Min),
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		)
		ch <- prometheus.MustNewConstMetric(
			kafkaWriterBatchSizeAvg,
			prometheus.GaugeValue,
			float64(stats.BatchSize.Avg),
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		)
		ch <- prometheus.MustNewConstMetric(
			kafkaWriterBatchSizeMax,
			prometheus.GaugeValue,
			float64(stats.BatchSize.Max),
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		)
	
		ch <- prometheus.MustNewConstMetric(
			kafkaWriterBatchBytesMin,
			prometheus.GaugeValue,
			float64(stats.BatchBytes.Min),
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		)
		ch <- prometheus.MustNewConstMetric(
			kafkaWriterBatchBytesAvg,
			prometheus.GaugeValue,
			float64(stats.BatchBytes.Avg),
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		)
		ch <- prometheus.MustNewConstMetric(
			kafkaWriterBatchBytesMax,
			prometheus.GaugeValue,
			float64(stats.BatchBytes.Max),
			desc.pipeline,
			desc.pluginName,
			desc.clientId,
		)
	}
}

func (c *kafkaWriterCollector) append(d writerDescriptor, f func() kafka.WriterStats) {
	c.statFuncs[d] = f
}

func (c *kafkaWriterCollector) delete(d writerDescriptor) {
	delete(c.statFuncs, d)
}

func RegisterKafkaWriter(pipeline, pluginName, clientId string, statFunc func() kafka.WriterStats) {
	kafkaWriterMetricsRegister.Do(func() {
		prometheus.MustRegister(kafkaWriterMetricsCollector)
		prometheus.MustRegister(kafkaWriterMessagesCount)
		prometheus.MustRegister(kafkaWriterBytesCount)
		prometheus.MustRegister(kafkaWriterErrorsCount)
		prometheus.MustRegister(kafkaWriterBatchSecondsCount)
		prometheus.MustRegister(kafkaWriterBatchSecondsSum)
		prometheus.MustRegister(kafkaWriterWriteSecondsCount)
		prometheus.MustRegister(kafkaWriterWriteSecondsSum)
		prometheus.MustRegister(kafkaWriterBatchQueueSecondsCount)
		prometheus.MustRegister(kafkaWriterBatchQueueSecondsSum)
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
