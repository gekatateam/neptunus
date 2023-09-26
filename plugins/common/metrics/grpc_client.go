package metrics

import (
	"context"
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
			Help:       "Total number of received stream messages.",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"pipeline", "plugin_name", "procedure", "type"},
	)

	grpcClientSentMsgSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "plugin_grpc_client_sent_messages_seconds",
			Help:       "Total number of sent stream messages.",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"pipeline", "plugin_name", "procedure", "type"},
	)
}

type clientStreamWrapper struct {
	grpc.ClientStream

	closed     bool
	pipeline   string
	pluginName string
	procedure  string
	gRPCType   gRPCType
	start      time.Time
}

func (w *clientStreamWrapper) CloseSend() error {
	err := w.ClientStream.CloseSend()
	if err == nil && !w.closed {
		w.closed = true
		grpcClientCallsSummary.WithLabelValues(
			w.pipeline, w.pluginName, w.procedure, string(w.gRPCType), fromGrpcError(err).Code().String(),
		).Observe(floatSeconds(w.start))

		grpcClientCompletedTotal.WithLabelValues(
			w.pipeline, w.pluginName, w.procedure, string(w.gRPCType),
		).Inc()
	}

	return err
}

func (w *clientStreamWrapper) RecvMsg(m any) error {
	begin := time.Now()

	err := w.ClientStream.RecvMsg(m)
	if err == nil {
		grpcClientReceivedMsgSummary.WithLabelValues(
			w.pipeline, w.pluginName, w.procedure, string(w.gRPCType),
		).Observe(floatSeconds(begin))
		return nil
	}

	if err != nil && !w.closed { // stream done
		w.closed = true
		grpcClientCallsSummary.WithLabelValues(
			w.pipeline, w.pluginName, w.procedure, string(w.gRPCType), fromGrpcError(err).Code().String(),
		).Observe(floatSeconds(w.start))

		grpcClientCompletedTotal.WithLabelValues(
			w.pipeline, w.pluginName, w.procedure, string(w.gRPCType),
		).Inc()
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

func GrpcClientStreamInterceptor(pipeline, pluginName string) grpc.StreamClientInterceptor {
	grpcClientMetricsRegister.Do(func() {
		prometheus.MustRegister(grpcClientCallsSummary)
		prometheus.MustRegister(grpcClientCalledTotal)
		prometheus.MustRegister(grpcClientCompletedTotal)
		prometheus.MustRegister(grpcClientReceivedMsgSummary)
		prometheus.MustRegister(grpcClientSentMsgSummary)
	})

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		begin := time.Now()
		procedure := method
		rpcType := clientStreamType(desc)

		grpcClientCalledTotal.WithLabelValues(
			pipeline, pluginName, procedure, string(rpcType),
		).Inc()

		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil { // stream open failed
			grpcClientCallsSummary.WithLabelValues(
				pipeline, pluginName, procedure, string(rpcType), fromGrpcError(err).Code().String(),
			).Observe(floatSeconds(begin))

			grpcClientCompletedTotal.WithLabelValues(
				pipeline, pluginName, procedure, string(rpcType),
			).Inc()

			return clientStream, err
		}

		return &clientStreamWrapper{
			clientStream,
			false,
			pipeline,
			pluginName,
			procedure,
			rpcType,
			begin,
		}, nil
	}
}

func GrpcClientUnaryInterceptor(pipeline, pluginName string) grpc.UnaryClientInterceptor {
	grpcClientMetricsRegister.Do(func() {
		prometheus.MustRegister(grpcClientCallsSummary)
		prometheus.MustRegister(grpcClientCalledTotal)
		prometheus.MustRegister(grpcClientCompletedTotal)
		prometheus.MustRegister(grpcClientReceivedMsgSummary)
		prometheus.MustRegister(grpcClientSentMsgSummary)
	})

	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		begin := time.Now()
		procedure := method
		rpcType := unary

		grpcClientCalledTotal.WithLabelValues(
			pipeline, pluginName, procedure, string(rpcType),
		).Inc()

		err := invoker(ctx, method, req, reply, cc, opts...)

		grpcClientCallsSummary.WithLabelValues(
			pipeline, pluginName, procedure, string(rpcType), fromGrpcError(err).Code().String(),
		).Observe(floatSeconds(begin))

		grpcClientCompletedTotal.WithLabelValues(
			pipeline, pluginName, procedure, string(rpcType),
		).Inc()

		return err
	}
}

func clientStreamType(info *grpc.StreamDesc) gRPCType {
	if info.ClientStreams && !info.ServerStreams {
		return clientStream
	} else if !info.ClientStreams && info.ServerStreams {
		return serverStream
	}
	return bidiStream
}
