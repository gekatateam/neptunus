package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type EventStatus string

var (
	EventAccepted EventStatus = "accepted"
	EventRejected EventStatus = "rejected"
	EventFailed   EventStatus = "failed"
)

var (
	inputSummary     *prometheus.SummaryVec
	filterSummary    *prometheus.SummaryVec
	processorSummary *prometheus.SummaryVec
	outputSummary    *prometheus.SummaryVec
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
		[]string{"plugin", "name", "status"},
	)

	filterSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "filter_plugin_processed_events",
			Help:       "Events statistic for filters",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"plugin", "name", "status"},
	)

	processorSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "processor_plugin_processed_events",
			Help:       "Events statistic for processors",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"plugin", "name", "status"},
	)

	outputSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "output_plugin_processed_events",
			Help:       "Events statistic for outputs",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"plugin", "name", "status"},
	)

	prometheus.MustRegister(inputSummary)
	prometheus.MustRegister(filterSummary)
	prometheus.MustRegister(processorSummary)
	prometheus.MustRegister(outputSummary)
}

func ObserveInputSummary(plugin, name string, status EventStatus, t time.Duration) {
	inputSummary.WithLabelValues(plugin, name, string(status)).Observe(float64(t) / float64(time.Second))
}

func ObserveFliterSummary(plugin, name string, status EventStatus, t time.Duration) {
	filterSummary.WithLabelValues(plugin, name, string(status)).Observe(float64(t) / float64(time.Second))
}

func ObserveProcessorSummary(plugin, name string, status EventStatus, t time.Duration) {
	processorSummary.WithLabelValues(plugin, name, string(status)).Observe(float64(t) / float64(time.Second))
}

func ObserveOutputSummary(plugin, name string, status EventStatus, t time.Duration) {
	outputSummary.WithLabelValues(plugin, name, string(status)).Observe(float64(t) / float64(time.Second))
}
