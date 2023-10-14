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
	coreSummary       *prometheus.SummaryVec
	inputSummary      *prometheus.SummaryVec
	filterSummary     *prometheus.SummaryVec
	processorSummary  *prometheus.SummaryVec
	outputSummary     *prometheus.SummaryVec
	parserSummary     *prometheus.SummaryVec
	serializerSummary *prometheus.SummaryVec

	chans        *chanCollector
	chanCapacity *prometheus.Desc
	chanLength   *prometheus.Desc

	pipes     *pipelineCollectror
	pipeState *prometheus.Desc
	pipeLines *prometheus.Desc
)

func init() {
	// plugins stats
	// status="(accepted|rejected|failed)"
	inputSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "input_plugin_processed_events",
			Help:       "Events statistic for inputs.",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001, 1.0: 0},
		},
		[]string{"plugin", "name", "pipeline", "status"},
	)

	filterSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "filter_plugin_processed_events",
			Help:       "Events statistic for filters.",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001, 1.0: 0},
		},
		[]string{"plugin", "name", "pipeline", "status"},
	)

	processorSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "processor_plugin_processed_events",
			Help:       "Events statistic for processors.",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001, 1.0: 0},
		},
		[]string{"plugin", "name", "pipeline", "status"},
	)

	outputSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "output_plugin_processed_events",
			Help:       "Events statistic for outputs.",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001, 1.0: 0},
		},
		[]string{"plugin", "name", "pipeline", "status"},
	)

	parserSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "parser_plugin_processed_events",
			Help:       "Events statistic for parsers.",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001, 1.0: 0},
		},
		[]string{"plugin", "name", "pipeline", "status"},
	)

	serializerSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "serializer_plugin_processed_events",
			Help:       "Events statistic for serializers.",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001, 1.0: 0},
		},
		[]string{"plugin", "name", "pipeline", "status"},
	)

	coreSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "core_plugin_processed_events",
			Help:       "Events statistic for core plugins.",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001, 1.0: 0},
		},
		[]string{"plugin", "name", "pipeline", "status"},
	)

	// unit-to-unit channels internal stats
	chans = &chanCollector{}
	chanLength = prometheus.NewDesc(
		"pipeline_channel_length",
		"Pipeline unit-to-unit channel communication length.",
		[]string{"pipeline", "inner", "outer"},
		nil,
	)
	chanCapacity = prometheus.NewDesc(
		"pipeline_channel_capacity",
		"Pipeline unit-to-unit channel communication capacity.",
		[]string{"pipeline", "inner", "outer"},
		nil,
	)

	// pipelines stats
	pipes = &pipelineCollectror{}
	pipeState = prometheus.NewDesc(
		"pipeline_state",
		"Pipeline state: 1-5 is for Created, Starting, Running, Stopping, Stopped.",
		[]string{"pipeline"},
		nil,
	)
	pipeLines = prometheus.NewDesc(
		"pipeline_processors_lines",
		"Number of configured processors lines.",
		[]string{"pipeline"},
		nil,
	)

	prometheus.MustRegister(inputSummary)
	prometheus.MustRegister(filterSummary)
	prometheus.MustRegister(processorSummary)
	prometheus.MustRegister(outputSummary)
	prometheus.MustRegister(parserSummary)
	prometheus.MustRegister(serializerSummary)
	prometheus.MustRegister(coreSummary)
	prometheus.MustRegister(chans)
	prometheus.MustRegister(pipes)
}

func ObserveInputSummary(plugin, name, pipeline string, status eventStatus, t time.Duration) {
	inputSummary.WithLabelValues(plugin, name, pipeline, string(status)).Observe(t.Seconds())
}

func ObserveFliterSummary(plugin, name, pipeline string, status eventStatus, t time.Duration) {
	filterSummary.WithLabelValues(plugin, name, pipeline, string(status)).Observe(t.Seconds())
}

func ObserveProcessorSummary(plugin, name, pipeline string, status eventStatus, t time.Duration) {
	processorSummary.WithLabelValues(plugin, pipeline, name, string(status)).Observe(t.Seconds())
}

func ObserveOutputSummary(plugin, name, pipeline string, status eventStatus, t time.Duration) {
	outputSummary.WithLabelValues(plugin, name, pipeline, string(status)).Observe(t.Seconds())
}

func ObserveParserSummary(plugin, name, pipeline string, status eventStatus, t time.Duration) {
	parserSummary.WithLabelValues(plugin, name, pipeline, string(status)).Observe(t.Seconds())
}

func ObserveSerializerSummary(plugin, name, pipeline string, status eventStatus, t time.Duration) {
	serializerSummary.WithLabelValues(plugin, name, pipeline, string(status)).Observe(t.Seconds())
}

func ObserveCoreSummary(plugin, name, pipeline string, status eventStatus, t time.Duration) {
	coreSummary.WithLabelValues(plugin, name, pipeline, string(status)).Observe(t.Seconds())
}

func CollectChan(statFunc func() ChanStats) {
	chans.append(statFunc)
}

func CollectPipes(statFunc func() map[string]struct {
	State int
	Lines int
}) {
	pipes.set(statFunc)
}
