package metrics

import (
	"fmt"
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
	CoreSet.GetOrCreateSummaryExt(
		fmt.Sprintf("input_plugin_processed_events{plugin=%q,name=%q,pipeline=%q,status=%q}", plugin, name, pipeline, status),
		DefaultMetricWindow,
		DefaultSummaryQuantiles,
	).Update(t.Seconds())
}

func ObserveFilterSummary(plugin, name, pipeline string, status EventStatus, t time.Duration) {
	CoreSet.GetOrCreateSummaryExt(
		fmt.Sprintf("filter_plugin_processed_events{plugin=%q,name=%q,pipeline=%q,status=%q}", plugin, name, pipeline, status),
		DefaultMetricWindow,
		DefaultSummaryQuantiles,
	).Update(t.Seconds())
}

func ObserveProcessorSummary(plugin, name, pipeline string, status EventStatus, t time.Duration) {
	CoreSet.GetOrCreateSummaryExt(
		fmt.Sprintf("processor_plugin_processed_events{plugin=%q,name=%q,pipeline=%q,status=%q}", plugin, name, pipeline, status),
		DefaultMetricWindow,
		DefaultSummaryQuantiles,
	).Update(t.Seconds())
}

func ObserveOutputSummary(plugin, name, pipeline string, status EventStatus, t time.Duration) {
	CoreSet.GetOrCreateSummaryExt(
		fmt.Sprintf("output_plugin_processed_events{plugin=%q,name=%q,pipeline=%q,status=%q}", plugin, name, pipeline, status),
		DefaultMetricWindow,
		DefaultSummaryQuantiles,
	).Update(t.Seconds())
}

func ObserveParserSummary(plugin, name, pipeline string, status EventStatus, t time.Duration) {
	CoreSet.GetOrCreateSummaryExt(
		fmt.Sprintf("parser_plugin_processed_events{plugin=%q,name=%q,pipeline=%q,status=%q}", plugin, name, pipeline, status),
		DefaultMetricWindow,
		DefaultSummaryQuantiles,
	).Update(t.Seconds())
}

func ObserveSerializerSummary(plugin, name, pipeline string, status EventStatus, t time.Duration) {
	CoreSet.GetOrCreateSummaryExt(
		fmt.Sprintf("serializer_plugin_processed_events{plugin=%q,name=%q,pipeline=%q,status=%q}", plugin, name, pipeline, status),
		DefaultMetricWindow,
		DefaultSummaryQuantiles,
	).Update(t.Seconds())
}

func ObserveCoreSummary(plugin, name, pipeline string, status EventStatus, t time.Duration) {
	CoreSet.GetOrCreateSummaryExt(
		fmt.Sprintf("core_plugin_processed_events{plugin=%q,name=%q,pipeline=%q,status=%q}", plugin, name, pipeline, status),
		DefaultMetricWindow,
		DefaultSummaryQuantiles,
	).Update(t.Seconds())
}
