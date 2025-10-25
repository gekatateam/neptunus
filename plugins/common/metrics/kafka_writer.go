package metrics

import (
	"fmt"
	"sync"

	"github.com/segmentio/kafka-go"

	"github.com/gekatateam/neptunus/metrics"
)

var (
	kafkaWriterMetricsRegister  = &sync.Once{}
	kafkaWriterMetricsCollector = &kafkaWriterCollector{
		statFuncs: make(map[writerDescriptor]func() kafka.WriterStats),
		mu:        &sync.Mutex{},
	}
)

var (
	kafkaWriterMessagesCount = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_messages_count{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterBytesCount = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_bytes_count{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterErrorsCount = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_errors_count{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}

	kafkaWriterBatchSecondsCount = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_seconds_count{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterBatchSecondsSum = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_seconds_sum{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterBatchSecondsMin = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_seconds_min{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterBatchSecondsAvg = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_seconds_avg{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterBatchSecondsMax = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_seconds_max{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}

	kafkaWriterWriteSecondsCount = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_write_seconds_count{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterWriteSecondsSum = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_write_seconds_sum{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterWriteSecondsMin = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_write_seconds_min{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterWriteSecondsAvg = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_write_seconds_avg{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterWriteSecondsMax = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_write_seconds_max{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}

	kafkaWriterBatchQueueSecondsCount = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_queue_seconds_count{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterBatchQueueSecondsSum = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_queue_seconds_sum{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterBatchQueueSecondsMin = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_queue_seconds_min{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterBatchQueueSecondsAvg = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_queue_seconds_avg{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterBatchQueueSecondsMax = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_queue_seconds_max{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}

	kafkaWriterBatchSizeCount = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_size_count{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterBatchSizeSum = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_size_sum{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterBatchSizeMin = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_size_min{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterBatchSizeAvg = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_size_avg{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterBatchSizeMax = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_size_max{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}

	kafkaWriterBatchBytesCount = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_bytes_count{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterBatchBytesSum = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_bytes_sum{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterBatchBytesMin = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_bytes_min{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterBatchBytesAvg = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_bytes_avg{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
	kafkaWriterBatchBytesMax = func(d writerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_writer_batch_bytes_max{pipeline=%q,plugin_name=%q,topic=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.clientId)
	}
)

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

	metrics.PluginsSet.UnregisterMetric(kafkaWriterMessagesCount(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterBytesCount(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterErrorsCount(d))

	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchSecondsCount(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchSecondsSum(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchSecondsMin(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchSecondsAvg(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchSecondsMax(d))

	metrics.PluginsSet.UnregisterMetric(kafkaWriterWriteSecondsCount(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterWriteSecondsSum(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterWriteSecondsMin(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterWriteSecondsAvg(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterWriteSecondsMax(d))

	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchQueueSecondsCount(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchQueueSecondsSum(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchQueueSecondsMin(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchQueueSecondsAvg(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchQueueSecondsMax(d))

	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchSizeCount(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchSizeSum(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchSizeMin(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchSizeAvg(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchSizeMax(d))

	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchBytesCount(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchBytesSum(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchBytesMin(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchBytesAvg(d))
	metrics.PluginsSet.UnregisterMetric(kafkaWriterBatchBytesMax(d))
}

func (c *kafkaWriterCollector) Collect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for d, f := range c.statFuncs {
		stats := f()

		// it is very important to know - all kafka.WriterStats are reset after collection
		// this is why all counters here updated by Add(), not by Set()
		metrics.PluginsSet.GetOrCreateCounter(kafkaWriterMessagesCount(d)).Add(int(stats.Messages))
		metrics.PluginsSet.GetOrCreateCounter(kafkaWriterBytesCount(d)).Add(int(stats.Bytes))
		metrics.PluginsSet.GetOrCreateCounter(kafkaWriterErrorsCount(d)).Add(int(stats.Errors))

		metrics.PluginsSet.GetOrCreateCounter(kafkaWriterBatchSecondsCount(d)).Add(int(stats.BatchTime.Count))
		metrics.PluginsSet.GetOrCreateFloatCounter(kafkaWriterBatchSecondsSum(d)).Add(stats.BatchTime.Sum.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaWriterBatchSecondsMin(d), nil).Set(stats.BatchTime.Min.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaWriterBatchSecondsAvg(d), nil).Set(stats.BatchTime.Avg.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaWriterBatchSecondsMax(d), nil).Set(stats.BatchTime.Max.Seconds())

		metrics.PluginsSet.GetOrCreateCounter(kafkaWriterWriteSecondsCount(d)).Add(int(stats.WriteTime.Count))
		metrics.PluginsSet.GetOrCreateFloatCounter(kafkaWriterWriteSecondsSum(d)).Add(stats.WriteTime.Sum.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaWriterWriteSecondsMin(d), nil).Set(stats.WriteTime.Min.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaWriterWriteSecondsAvg(d), nil).Set(stats.WriteTime.Avg.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaWriterWriteSecondsMax(d), nil).Set(stats.WriteTime.Max.Seconds())

		metrics.PluginsSet.GetOrCreateCounter(kafkaWriterBatchQueueSecondsCount(d)).Add(int(stats.BatchQueueTime.Count))
		metrics.PluginsSet.GetOrCreateFloatCounter(kafkaWriterBatchQueueSecondsSum(d)).Add(stats.BatchQueueTime.Sum.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaWriterBatchQueueSecondsMin(d), nil).Set(stats.BatchQueueTime.Min.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaWriterBatchQueueSecondsAvg(d), nil).Set(stats.BatchQueueTime.Avg.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaWriterBatchQueueSecondsMax(d), nil).Set(stats.BatchQueueTime.Max.Seconds())

		metrics.PluginsSet.GetOrCreateCounter(kafkaWriterBatchSizeCount(d)).Add(int(stats.BatchSize.Count))
		metrics.PluginsSet.GetOrCreateCounter(kafkaWriterBatchSizeSum(d)).Add(int(stats.BatchSize.Sum))
		metrics.PluginsSet.GetOrCreateGauge(kafkaWriterBatchSizeMin(d), nil).Set(float64(stats.BatchSize.Min))
		metrics.PluginsSet.GetOrCreateGauge(kafkaWriterBatchSizeAvg(d), nil).Set(float64(stats.BatchSize.Avg))
		metrics.PluginsSet.GetOrCreateGauge(kafkaWriterBatchSizeMax(d), nil).Set(float64(stats.BatchSize.Max))

		metrics.PluginsSet.GetOrCreateCounter(kafkaWriterBatchBytesCount(d)).Add(int(stats.BatchBytes.Count))
		metrics.PluginsSet.GetOrCreateCounter(kafkaWriterBatchBytesSum(d)).Add(int(stats.BatchBytes.Sum))
		metrics.PluginsSet.GetOrCreateGauge(kafkaWriterBatchBytesMin(d), nil).Set(float64(stats.BatchBytes.Min))
		metrics.PluginsSet.GetOrCreateGauge(kafkaWriterBatchBytesAvg(d), nil).Set(float64(stats.BatchBytes.Avg))
		metrics.PluginsSet.GetOrCreateGauge(kafkaWriterBatchBytesMax(d), nil).Set(float64(stats.BatchBytes.Max))
	}
}

func RegisterKafkaWriter(pipeline, pluginName, topic, clientId string, statFunc func() kafka.WriterStats) {
	kafkaWriterMetricsRegister.Do(func() {
		metrics.GlobalCollectorsRunner.Append(kafkaWriterMetricsCollector)
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
