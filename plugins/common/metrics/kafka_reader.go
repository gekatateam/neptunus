package metrics

import (
	"fmt"
	"sync"

	"github.com/segmentio/kafka-go"

	"github.com/gekatateam/neptunus/metrics"
)

var (
	kafkaReaderMetricsRegister  = &sync.Once{}
	kafkaReaderMetricsCollector = &kafkaReaderCollector{
		statFuncs: make(map[readerDescriptor]func() ReaderStats),
		mu:        &sync.Mutex{},
	}
)

var (
	kafkaReaderMessagesCount = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_messages_count{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderBytesCount = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_bytes_count{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderErrorsCount = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_errors_count{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderTimeoutsCount = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_timeouts_count{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderFetchesCount = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_fetches_count{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}

	kafkaReaderOffset = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_offset{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderLag = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_lag{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderDelay = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_delay{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderCommitQueueCapacity = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_commit_queue_capacity{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderCommitQueueLength = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_commit_queue_length{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}

	kafkaReaderDialSecondsCount = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_dial_seconds_count{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderDialSecondsSum = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_dial_seconds_sum{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderDialSecondsMin = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_dial_seconds_min{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderDialSecondsAvg = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_dial_seconds_avg{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderDialSecondsMax = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_dial_seconds_max{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}

	kafkaReaderReadSecondsCount = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_read_seconds_count{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderReadSecondsSum = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_read_seconds_sum{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderReadSecondsMin = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_read_seconds_min{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderReadSecondsAvg = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_read_seconds_avg{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderReadSecondsMax = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_read_seconds_max{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}

	kafkaReaderWaitSecondsCount = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_wait_seconds_count{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderWaitSecondsSum = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_wait_seconds_sum{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderWaitSecondsMin = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_wait_seconds_min{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderWaitSecondsAvg = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_wait_seconds_avg{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderWaitSecondsMax = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_wait_seconds_max{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}

	kafkaReaderFetchSizeCount = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_fetch_size_count{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderFetchSizeSum = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_fetch_size_sum{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderFetchSizeMin = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_fetch_size_min{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderFetchSizeAvg = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_fetch_size_avg{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderFetchSizeMax = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_fetch_size_max{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}

	kafkaReaderFetchBytesCount = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_fetch_bytes_count{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderFetchBytesSum = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_fetch_bytes_sum{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderFetchBytesMin = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_fetch_bytes_min{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderFetchBytesAvg = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_fetch_bytes_avg{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
	kafkaReaderFetchBytesMax = func(d readerDescriptor) string {
		return fmt.Sprintf("plugin_kafka_reader_fetch_bytes_max{pipeline=%q,plugin_name=%q,topic=%q,partition=%q,group_id=%q,client_id=%q}", d.pipeline, d.pluginName, d.topic, d.partition, d.groupId, d.clientId)
	}
)

type ReaderStats struct {
	kafka.ReaderStats
	CommitQueueLength   int
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

	metrics.PluginsSet.UnregisterMetric(kafkaReaderMessagesCount(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderBytesCount(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderErrorsCount(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderFetchesCount(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderTimeoutsCount(d))

	metrics.PluginsSet.UnregisterMetric(kafkaReaderOffset(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderLag(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderDelay(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderCommitQueueCapacity(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderCommitQueueLength(d))

	metrics.PluginsSet.UnregisterMetric(kafkaReaderDialSecondsCount(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderDialSecondsSum(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderDialSecondsAvg(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderDialSecondsMin(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderDialSecondsMax(d))

	metrics.PluginsSet.UnregisterMetric(kafkaReaderReadSecondsCount(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderReadSecondsSum(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderReadSecondsAvg(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderReadSecondsMin(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderReadSecondsMax(d))

	metrics.PluginsSet.UnregisterMetric(kafkaReaderWaitSecondsCount(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderWaitSecondsSum(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderWaitSecondsAvg(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderWaitSecondsMin(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderWaitSecondsMax(d))

	metrics.PluginsSet.UnregisterMetric(kafkaReaderFetchSizeCount(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderFetchSizeSum(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderFetchSizeMin(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderFetchSizeAvg(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderFetchSizeMax(d))

	metrics.PluginsSet.UnregisterMetric(kafkaReaderFetchBytesCount(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderFetchBytesSum(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderFetchBytesMin(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderFetchBytesAvg(d))
	metrics.PluginsSet.UnregisterMetric(kafkaReaderFetchBytesMax(d))
}

func (c *kafkaReaderCollector) Collect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for d, f := range c.statFuncs {
		stats := f()

		// it is very important to know - all kafka.ReaderStats are reset after collection
		// this is why all counters here updated by Add(), not by Set()
		metrics.PluginsSet.GetOrCreateCounter(kafkaReaderMessagesCount(d)).Add(int(stats.Messages))
		metrics.PluginsSet.GetOrCreateCounter(kafkaReaderBytesCount(d)).Add(int(stats.Bytes))
		metrics.PluginsSet.GetOrCreateCounter(kafkaReaderErrorsCount(d)).Add(int(stats.Errors))
		metrics.PluginsSet.GetOrCreateCounter(kafkaReaderFetchesCount(d)).Add(int(stats.Fetches))
		metrics.PluginsSet.GetOrCreateCounter(kafkaReaderTimeoutsCount(d)).Add(int(stats.Timeouts))

		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderOffset(d), nil).Set(float64(stats.Offset))
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderLag(d), nil).Set(float64(stats.Lag))
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderDelay(d), nil).Set(float64(stats.Delay))
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderCommitQueueCapacity(d), nil).Set(float64(stats.CommitQueueCapacity))
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderCommitQueueLength(d), nil).Set(float64(stats.CommitQueueLength))

		metrics.PluginsSet.GetOrCreateCounter(kafkaReaderDialSecondsCount(d)).Add(int(stats.DialTime.Count))
		metrics.PluginsSet.GetOrCreateFloatCounter(kafkaReaderDialSecondsSum(d)).Add(stats.DialTime.Sum.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderDialSecondsAvg(d), nil).Set(stats.DialTime.Min.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderDialSecondsMin(d), nil).Set(stats.DialTime.Avg.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderDialSecondsMax(d), nil).Set(stats.DialTime.Max.Seconds())

		metrics.PluginsSet.GetOrCreateCounter(kafkaReaderReadSecondsCount(d)).Add(int(stats.ReadTime.Count))
		metrics.PluginsSet.GetOrCreateFloatCounter(kafkaReaderReadSecondsSum(d)).Add(stats.ReadTime.Sum.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderReadSecondsAvg(d), nil).Set(stats.ReadTime.Min.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderReadSecondsMin(d), nil).Set(stats.ReadTime.Avg.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderReadSecondsMax(d), nil).Set(stats.ReadTime.Max.Seconds())

		metrics.PluginsSet.GetOrCreateCounter(kafkaReaderWaitSecondsCount(d)).Add(int(stats.WaitTime.Count))
		metrics.PluginsSet.GetOrCreateFloatCounter(kafkaReaderWaitSecondsSum(d)).Add(stats.WaitTime.Sum.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderWaitSecondsAvg(d), nil).Set(stats.WaitTime.Min.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderWaitSecondsMin(d), nil).Set(stats.WaitTime.Avg.Seconds())
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderWaitSecondsMax(d), nil).Set(stats.WaitTime.Max.Seconds())

		metrics.PluginsSet.GetOrCreateCounter(kafkaReaderFetchSizeCount(d)).Add(int(stats.FetchSize.Count))
		metrics.PluginsSet.GetOrCreateCounter(kafkaReaderFetchSizeSum(d)).Add(int(stats.FetchSize.Sum))
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderFetchSizeMin(d), nil).Set(float64(stats.FetchSize.Min))
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderFetchSizeAvg(d), nil).Set(float64(stats.FetchSize.Avg))
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderFetchSizeMax(d), nil).Set(float64(stats.FetchSize.Max))

		metrics.PluginsSet.GetOrCreateCounter(kafkaReaderFetchBytesCount(d)).Add(int(stats.FetchBytes.Count))
		metrics.PluginsSet.GetOrCreateCounter(kafkaReaderFetchBytesSum(d)).Add(int(stats.FetchBytes.Sum))
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderFetchBytesMin(d), nil).Set(float64(stats.FetchBytes.Min))
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderFetchBytesAvg(d), nil).Set(float64(stats.FetchBytes.Avg))
		metrics.PluginsSet.GetOrCreateGauge(kafkaReaderFetchBytesMax(d), nil).Set(float64(stats.FetchBytes.Max))
	}
}

func RegisterKafkaReader(pipeline, pluginName, topic, partition, groupId, clientId string, statFunc func() ReaderStats) {
	kafkaReaderMetricsRegister.Do(func() {
		metrics.GlobalCollectorsRunner.Append(kafkaReaderMetricsCollector)
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
