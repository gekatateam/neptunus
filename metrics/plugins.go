package metrics

import (
	"time"
)

type EventStatus string

const (
	EventAccepted EventStatus = "accepted"
	EventRejected EventStatus = "rejected"
	EventFailed   EventStatus = "failed"
)

type ObserveFunc func(plugin, name, pipeline string, status EventStatus, t time.Duration)

func ObserveMock(plugin, name, pipeline string, status EventStatus, t time.Duration) {}

func ObserveInputSummary(plugin, name, pipeline string, status EventStatus, t time.Duration) {
	inputSummary.WithLabelValues(plugin, name, pipeline, string(status)).Observe(t.Seconds())
}

func ObserveFilterSummary(plugin, name, pipeline string, status EventStatus, t time.Duration) {
	filterSummary.WithLabelValues(plugin, name, pipeline, string(status)).Observe(t.Seconds())
}

func ObserveProcessorSummary(plugin, name, pipeline string, status EventStatus, t time.Duration) {
	processorSummary.WithLabelValues(plugin, name, pipeline, string(status)).Observe(t.Seconds())
}

func ObserveOutputSummary(plugin, name, pipeline string, status EventStatus, t time.Duration) {
	outputSummary.WithLabelValues(plugin, name, pipeline, string(status)).Observe(t.Seconds())
}

func ObserveParserSummary(plugin, name, pipeline string, status EventStatus, t time.Duration) {
	parserSummary.WithLabelValues(plugin, name, pipeline, string(status)).Observe(t.Seconds())
}

func ObserveSerializerSummary(plugin, name, pipeline string, status EventStatus, t time.Duration) {
	serializerSummary.WithLabelValues(plugin, name, pipeline, string(status)).Observe(t.Seconds())
}

func ObserveCoreSummary(plugin, name, pipeline string, status EventStatus, t time.Duration) {
	coreSummary.WithLabelValues(plugin, name, pipeline, string(status)).Observe(t.Seconds())
}
