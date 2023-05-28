package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type eventStatus string

const (
	EventAccepted eventStatus = "accepted"
	EventRejected eventStatus = "rejected"
	EventFailed   eventStatus = "failed"
)

var (
	coreSummary      *prometheus.SummaryVec
	inputSummary     *prometheus.SummaryVec
	filterSummary    *prometheus.SummaryVec
	processorSummary *prometheus.SummaryVec
	outputSummary    *prometheus.SummaryVec

	chans        *chanCollector
	chanCapacity *prometheus.Desc
	chanLength   *prometheus.Desc
)

func init() {
	// plugins stats
	// status="(accepted|rejected|failed)"
	inputSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "input_plugin_processed_events",
			Help:       "Events statistic for inputs",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"plugin", "name", "pipeline", "status"},
	)

	filterSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "filter_plugin_processed_events",
			Help:       "Events statistic for filters",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"plugin", "name", "pipeline", "status"},
	)

	processorSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "processor_plugin_processed_events",
			Help:       "Events statistic for processors",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"plugin", "name", "pipeline", "status"},
	)

	outputSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "output_plugin_processed_events",
			Help:       "Events statistic for outputs",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"plugin", "name", "pipeline", "status"},
	)

	coreSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "core_plugin_processed_events",
			Help:       "Events statistic for inputs",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"plugin", "name", "pipeline", "status"},
	)

	// unit-to-unit channels internal stats
	chans = &chanCollector{}
	chanLength = prometheus.NewDesc(
		"pipeline_channel_length",
		"Pipeline unit-to-unit channel communication length",
		[]string{"pipeline", "inner", "outer"},
		nil,
	)
	chanCapacity = prometheus.NewDesc(
		"pipeline_channel_capacity",
		"Pipeline unit-to-unit channel communication capacity",
		[]string{"pipeline", "inner", "outer"},
		nil,
	)

	prometheus.MustRegister(inputSummary)
	prometheus.MustRegister(filterSummary)
	prometheus.MustRegister(processorSummary)
	prometheus.MustRegister(outputSummary)
	prometheus.MustRegister(coreSummary)
	prometheus.MustRegister(chans)
}

func ObserveInputSummary(plugin, name, pipeline string, status eventStatus, t time.Duration) {
	inputSummary.WithLabelValues(plugin, name, pipeline, string(status)).Observe(float64(t) / float64(time.Second))
}

func ObserveFliterSummary(plugin, name, pipeline string, status eventStatus, t time.Duration) {
	filterSummary.WithLabelValues(plugin, name, pipeline, string(status)).Observe(float64(t) / float64(time.Second))
}

func ObserveProcessorSummary(plugin, name, pipeline string, status eventStatus, t time.Duration) {
	processorSummary.WithLabelValues(plugin, pipeline, name, string(status)).Observe(float64(t) / float64(time.Second))
}

func ObserveOutputSummary(plugin, name, pipeline string, status eventStatus, t time.Duration) {
	outputSummary.WithLabelValues(plugin, name, pipeline, string(status)).Observe(float64(t) / float64(time.Second))
}

func ObserveCoreSummary(plugin, name, pipeline string, status eventStatus, t time.Duration) {
	coreSummary.WithLabelValues(plugin, name, pipeline, string(status)).Observe(float64(t) / float64(time.Second))
}

func CollectChan(statFunc func() ChanStats) {
	chans.append(statFunc)
}
