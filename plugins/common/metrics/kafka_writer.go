package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
)

var (
	kafkaWriterMetricsRegister = &sync.Once{}
)

type kafkaWriterCollector struct {
	statFuncs map[string]func() kafka.WriterStats
}

func (*kafkaWriterCollector) Collect(ch chan<- prometheus.Metric)
func (*kafkaWriterCollector) Describe(ch chan<- *prometheus.Desc)
func (*kafkaWriterCollector) append(f func() kafka.WriterStats)

func RegisterKafkaWriter(clientId string, statFunc func() kafka.WriterStats)

func UnregisterKafkaWriter(clientId string)
