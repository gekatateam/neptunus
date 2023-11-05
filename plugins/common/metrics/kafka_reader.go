package metrics

import "github.com/segmentio/kafka-go"

func RegisterKafkaReader(pipeline, pluginName, topic, groupId, clientId string, statFunc func() kafka.ReaderStats) {

}

func UnregisterKafkaReader(pipeline, pluginName, topic, groupId, clientId string) {

}
