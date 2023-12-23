package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
)

var (
	kafkaWriterMetricsRegister  = &sync.Once{}
	kafkaWriterMetricsCollector = &kafkaWriterCollector{
		statFuncs: make(map[writerDescriptor]func() kafka.WriterStats),
		mu:        &sync.Mutex{},
	}

	kafkaWriterMessagesCount *prometheus.CounterVec
	kafkaWriterBytesCount    *prometheus.CounterVec
	kafkaWriterErrorsCount   *prometheus.CounterVec

	kafkaWriterWriteSecondsCount *prometheus.CounterVec
	kafkaWriterWriteSecondsSum   *prometheus.CounterVec
	kafkaWriterWriteSecondsMin   *prometheus.GaugeVec
	kafkaWriterWriteSecondsAvg   *prometheus.GaugeVec
	kafkaWriterWriteSecondsMax   *prometheus.GaugeVec

	kafkaWriterBatchSecondsCount *prometheus.CounterVec
	kafkaWriterBatchSecondsSum   *prometheus.CounterVec
	kafkaWriterBatchSecondsMin   *prometheus.GaugeVec
	kafkaWriterBatchSecondsAvg   *prometheus.GaugeVec
	kafkaWriterBatchSecondsMax   *prometheus.GaugeVec

	kafkaWriterBatchQueueSecondsCount *prometheus.CounterVec
	kafkaWriterBatchQueueSecondsSum   *prometheus.CounterVec
	kafkaWriterBatchQueueSecondsMin   *prometheus.GaugeVec
	kafkaWriterBatchQueueSecondsAvg   *prometheus.GaugeVec
	kafkaWriterBatchQueueSecondsMax   *prometheus.GaugeVec

	kafkaWriterBatchSizeCount *prometheus.CounterVec
	kafkaWriterBatchSizeSum   *prometheus.CounterVec
	kafkaWriterBatchSizeMin   *prometheus.GaugeVec
	kafkaWriterBatchSizeAvg   *prometheus.GaugeVec
	kafkaWriterBatchSizeMax   *prometheus.GaugeVec

	kafkaWriterBatchBytesCount *prometheus.CounterVec
	kafkaWriterBatchBytesSum   *prometheus.CounterVec
	kafkaWriterBatchBytesMin   *prometheus.GaugeVec
	kafkaWriterBatchBytesAvg   *prometheus.GaugeVec
	kafkaWriterBatchBytesMax   *prometheus.GaugeVec
)

func init() {
	kafkaWriterMessagesCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_messages_count",
			Help: "Number of messages written by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterBytesCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_bytes_count",
			Help: "Number of bytes written by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterErrorsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_errors_count",
			Help: "Number of errors occurred during writes",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)

	kafkaWriterBatchSecondsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_batch_seconds_count",
			Help: "Number of batches created by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterBatchSecondsSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_batch_seconds_sum",
			Help: "Total time spent on filling batches",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterBatchSecondsMin = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_writer_batch_seconds_min",
			Help: "Min time spent on filling batches",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterBatchSecondsAvg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_writer_batch_seconds_avg",
			Help: "Average time spent on filling batches",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterBatchSecondsMax = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_writer_batch_seconds_max",
			Help: "Max time spent on filling batches",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)

	kafkaWriterWriteSecondsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_write_seconds_count",
			Help: "Number of writes performed by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterWriteSecondsSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_write_seconds_sum",
			Help: "Total time spent on writes",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterWriteSecondsMin = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_writer_write_seconds_min",
			Help: "Min time spent on writes",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterWriteSecondsAvg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_writer_write_seconds_avg",
			Help: "Average time spent on writes",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterWriteSecondsMax = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_writer_write_seconds_max",
			Help: "Max time spent on writes",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)

	kafkaWriterBatchQueueSecondsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_batch_queue_seconds_count",
			Help: "Number of batches placed in queue",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterBatchQueueSecondsSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_batch_queue_seconds_sum",
			Help: "Total time spent in queue",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterBatchQueueSecondsMin = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_writer_batch_queue_seconds_min",
			Help: "Min time spent in queue",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterBatchQueueSecondsAvg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_writer_batch_queue_seconds_avg",
			Help: "Average time spent in queue",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterBatchQueueSecondsMax = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_writer_batch_queue_seconds_max",
			Help: "Max time spent in queue",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)

	kafkaWriterBatchSizeCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_batch_size_count",
			Help: "Number of batches written by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterBatchSizeSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_batch_size_sum",
			Help: "Total size of written batches",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterBatchSizeMin = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_writer_batch_size_min",
			Help: "Min written batch size",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterBatchSizeAvg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_writer_batch_size_avg",
			Help: "Average written batch size",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterBatchSizeMax = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_writer_batch_size_max",
			Help: "Max written batch size",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)

	kafkaWriterBatchBytesCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_batch_bytes_count",
			Help: "Number of batches written by client",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterBatchBytesSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_kafka_writer_batch_bytes_sum",
			Help: "Total bytes of written batches",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterBatchBytesMin = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_writer_batch_bytes_min",
			Help: "Min written batch bytes",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterBatchBytesAvg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_writer_batch_bytes_avg",
			Help: "Average written batch bytes",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
	kafkaWriterBatchBytesMax = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugin_kafka_writer_batch_bytes_max",
			Help: "Max written batch bytes",
		},
		[]string{"pipeline", "plugin_name", "topic", "client_id"},
	)
}

type writerDescriptor struct {
	pipeline   string
	pluginName string
	topic      string
	clientId   string
}

type kafkaWriterCollector struct {
	statFuncs map[writerDescriptor]func() kafka.WriterStats
	mu        *sync.Mutex
}

func (c *kafkaWriterCollector) append(d writerDescriptor, f func() kafka.WriterStats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.statFuncs[d] = f
}

func (c *kafkaWriterCollector) delete(d writerDescriptor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.statFuncs, d)
}

func (c *kafkaWriterCollector) collect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for desc, f := range c.statFuncs {
		stats := f()

		kafkaWriterMessagesCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Add(float64(stats.Messages))
		kafkaWriterBytesCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Add(float64(stats.Bytes))
		kafkaWriterErrorsCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Add(float64(stats.Errors))

		kafkaWriterBatchSecondsCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Add(float64(stats.BatchTime.Count))
		kafkaWriterBatchSecondsSum.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Add(stats.BatchTime.Sum.Seconds())
		kafkaWriterBatchSecondsMin.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Set(stats.BatchTime.Min.Seconds())
		kafkaWriterBatchSecondsAvg.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Set(stats.BatchTime.Avg.Seconds())
		kafkaWriterBatchSecondsMax.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Set(stats.BatchTime.Max.Seconds())

		kafkaWriterWriteSecondsCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Add(float64(stats.WriteTime.Count))
		kafkaWriterWriteSecondsSum.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Add(stats.WriteTime.Sum.Seconds())
		kafkaWriterWriteSecondsMin.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Set(stats.WriteTime.Min.Seconds())
		kafkaWriterWriteSecondsAvg.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Set(stats.WriteTime.Avg.Seconds())
		kafkaWriterWriteSecondsMin.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Set(stats.WriteTime.Max.Seconds())

		kafkaWriterBatchQueueSecondsCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Add(float64(stats.BatchQueueTime.Count))
		kafkaWriterBatchQueueSecondsSum.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Add(stats.BatchQueueTime.Sum.Seconds())
		kafkaWriterBatchQueueSecondsMin.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Set(stats.BatchQueueTime.Min.Seconds())
		kafkaWriterBatchQueueSecondsAvg.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Set(stats.BatchQueueTime.Avg.Seconds())
		kafkaWriterBatchQueueSecondsMax.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Set(stats.BatchQueueTime.Max.Seconds())

		kafkaWriterBatchSizeCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Add(float64(stats.BatchSize.Count))
		kafkaWriterBatchSizeSum.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Add(float64(stats.BatchSize.Sum))
		kafkaWriterBatchSizeMin.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Set(float64(stats.BatchSize.Min))
		kafkaWriterBatchSizeAvg.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Set(float64(stats.BatchSize.Avg))
		kafkaWriterBatchSizeMax.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Set(float64(stats.BatchSize.Max))

		kafkaWriterBatchBytesCount.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Add(float64(stats.BatchBytes.Count))
		kafkaWriterBatchBytesSum.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Add(float64(stats.BatchBytes.Sum))
		kafkaWriterBatchBytesMin.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Set(float64(stats.BatchBytes.Min))
		kafkaWriterBatchBytesAvg.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Set(float64(stats.BatchBytes.Avg))
		kafkaWriterBatchBytesMax.WithLabelValues(
			desc.pipeline, desc.pluginName,
			desc.topic, desc.clientId,
		).Set(float64(stats.BatchBytes.Max))
	}
}

func RegisterKafkaWriter(pipeline, pluginName, topic, clientId string, statFunc func() kafka.WriterStats) {
	kafkaWriterMetricsRegister.Do(func() {
		prometheus.MustRegister(kafkaWriterMessagesCount)
		prometheus.MustRegister(kafkaWriterBytesCount)
		prometheus.MustRegister(kafkaWriterErrorsCount)

		prometheus.MustRegister(kafkaWriterWriteSecondsCount)
		prometheus.MustRegister(kafkaWriterWriteSecondsSum)
		prometheus.MustRegister(kafkaWriterWriteSecondsMin)
		prometheus.MustRegister(kafkaWriterWriteSecondsAvg)
		prometheus.MustRegister(kafkaWriterWriteSecondsMax)

		prometheus.MustRegister(kafkaWriterBatchSecondsCount)
		prometheus.MustRegister(kafkaWriterBatchSecondsSum)
		prometheus.MustRegister(kafkaWriterBatchSecondsMin)
		prometheus.MustRegister(kafkaWriterBatchSecondsAvg)
		prometheus.MustRegister(kafkaWriterBatchSecondsMax)

		prometheus.MustRegister(kafkaWriterBatchQueueSecondsCount)
		prometheus.MustRegister(kafkaWriterBatchQueueSecondsSum)
		prometheus.MustRegister(kafkaWriterBatchQueueSecondsMin)
		prometheus.MustRegister(kafkaWriterBatchQueueSecondsAvg)
		prometheus.MustRegister(kafkaWriterBatchQueueSecondsMax)

		prometheus.MustRegister(kafkaWriterBatchSizeCount)
		prometheus.MustRegister(kafkaWriterBatchSizeSum)
		prometheus.MustRegister(kafkaWriterBatchSizeMin)
		prometheus.MustRegister(kafkaWriterBatchSizeAvg)
		prometheus.MustRegister(kafkaWriterBatchSizeMax)

		prometheus.MustRegister(kafkaWriterBatchBytesCount)
		prometheus.MustRegister(kafkaWriterBatchBytesSum)
		prometheus.MustRegister(kafkaWriterBatchBytesMin)
		prometheus.MustRegister(kafkaWriterBatchBytesAvg)
		prometheus.MustRegister(kafkaWriterBatchBytesMax)

		t := time.NewTicker(15 * time.Second)
		go func() {
			for range t.C {
				kafkaWriterMetricsCollector.collect()
			}
		}()
	})

	kafkaWriterMetricsCollector.append(writerDescriptor{
		pipeline:   pipeline,
		pluginName: pluginName,
		topic:      topic,
		clientId:   clientId,
	}, statFunc)
}

func UnregisterKafkaWriter(pipeline, pluginName, topic, clientId string) {
	kafkaWriterMetricsCollector.delete(writerDescriptor{
		pipeline:   pipeline,
		pluginName: pluginName,
		topic:      topic,
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
