package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

var (
	grpcClientMetricsRegister    = &sync.Once{}
	grpcClientCalledTotal        *prometheus.CounterVec
	grpcClientCompletedTotal     *prometheus.CounterVec
	grpcClientCallsSummary       *prometheus.SummaryVec
	grpcClientReceivedMsgSummary *prometheus.SummaryVec
	grpcClientSentMsgSummary     *prometheus.SummaryVec
)

func init() {
	grpcClientCalledTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_grpc_client_called_total",
			Help: "Total number of started RPCs.",
		},
		[]string{"pipeline", "plugin_name", "procedure", "type"},
	)

	grpcClientCompletedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_grpc_client_completed_total",
			Help: "Total number of completed RPCs.",
		},
		[]string{"pipeline", "plugin_name", "procedure", "type"},
	)

	grpcClientCallsSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "plugin_grpc_client_calls_seconds",
			Help:       "Handled RPCs stats.",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"pipeline", "plugin_name", "procedure", "type", "status"},
	)

	grpcClientReceivedMsgSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "plugin_grpc_client_received_messages_seconds",
			Help:       "Total number of received messages.",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"pipeline", "plugin_name", "procedure", "type"},
	)

	grpcClientSentMsgSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "plugin_grpc_client_sent_messages_seconds",
			Help:       "Total number of sent messages.",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"pipeline", "plugin_name", "procedure", "type"},
	)
}

type clientStreamWrapper struct {
	grpc.ClientStream

	pipeline   string
	pluginName string
	procedure  string
	gRPCType   gRPCType
}

func (w *clientStreamWrapper) RecvMsg(m any) error {
	begin := time.Now()

	err := w.ClientStream.RecvMsg(m)
	if err == nil {
		grpcClientReceivedMsgSummary.WithLabelValues(
			w.pipeline, w.pluginName, w.procedure, string(w.gRPCType),
		).Observe(floatSeconds(begin))
	}

	return err
}

func (w *clientStreamWrapper) SendMsg(m any) error {
	begin := time.Now()

	err := w.ClientStream.SendMsg(m)
	if err == nil {
		grpcClientSentMsgSummary.WithLabelValues(
			w.pipeline, w.pluginName, w.procedure, string(w.gRPCType),
		).Observe(floatSeconds(begin))
	}

	return err
}



func clientStreamType(info *grpc.StreamDesc) gRPCType {
	if info.ClientStreams && !info.ServerStreams {
		return clientStream
	} else if !info.ClientStreams && info.ServerStreams {
		return serverStream
	}
	return bidiStream
}
