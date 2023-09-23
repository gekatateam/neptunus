package metrics

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	grpcServerMetricsRegister  = &sync.Once{}
	grpcServerCallsSummary     *prometheus.SummaryVec
	grpcServerCalledTotal      *prometheus.CounterVec
	grpcServerCompletedTotal   *prometheus.CounterVec
	grpcServerReceivedMsgTotal *prometheus.CounterVec
	grpcServerSentMsgTotal     *prometheus.CounterVec
)

func init() {
	grpcServerCallsSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "plugin_grpc_server_calls_seconds",
			Help:       "Handled RPCs stats.",
			MaxAge:     time.Minute,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"pipeline", "plugin_name", "procedure", "type", "status"},
	)

	grpcServerCalledTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_grpc_server_called_total",
			Help: "Total number of started RPCs.",
		},
		[]string{"pipeline", "plugin_name", "procedure", "type"},
	)

	grpcServerCompletedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_grpc_server_completed_total",
			Help: "Total number of completed RPCs.",
		},
		[]string{"pipeline", "plugin_name", "procedure", "type"},
	)

	grpcServerReceivedMsgTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_grpc_server_received_messages_total",
			Help: "Total number of received messages.",
		},
		[]string{"pipeline", "plugin_name", "procedure", "type"},
	)

	grpcServerSentMsgTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_grpc_server_sent_messages_total",
			Help: "Total number of sent messages.",
		},
		[]string{"pipeline", "plugin_name", "procedure", "type"},
	)
}

type gRPCType string

const (
	unary        gRPCType = "unary"
	serverStream gRPCType = "server_stream"
	clientStream gRPCType = "client_stream"
	bidiStream   gRPCType = "bidi_stream"
)

type streamWrapper struct {
	grpc.ServerStream

	pipeline   string
	pluginName string
	procedure  string
	gRPCType   gRPCType
}

func (w *streamWrapper) RecvMsg(m any) error {
	err := w.ServerStream.RecvMsg(m)
	if err != io.EOF {
		grpcServerReceivedMsgTotal.WithLabelValues(
			w.pipeline, w.pluginName, w.procedure, string(w.gRPCType),
		).Inc()
	}
	return err
}

func (w *streamWrapper) SendMsg(m any) error {
	err := w.ServerStream.SendMsg(m)
	if err != io.EOF {
		grpcServerSentMsgTotal.WithLabelValues(
			w.pipeline, w.pluginName, w.procedure, string(w.gRPCType),
		).Inc()
	}
	return err
}

func GrpcServerStreamInterceptor(pipeline, pluginName string) grpc.StreamServerInterceptor {
	grpcServerMetricsRegister.Do(func() {
		prometheus.MustRegister(grpcServerCallsSummary)
		prometheus.MustRegister(grpcServerCalledTotal)
		prometheus.MustRegister(grpcServerCompletedTotal)
		prometheus.MustRegister(grpcServerReceivedMsgTotal)
		prometheus.MustRegister(grpcServerSentMsgTotal)
	})

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		begin := time.Now()
		procedure := info.FullMethod
		rpcType := serverStreamType(info)

		grpcServerCalledTotal.WithLabelValues(
			pipeline, pluginName, procedure, string(rpcType),
		).Inc()

		err := handler(srv, &streamWrapper{ServerStream: ss, procedure: procedure, gRPCType: rpcType})

		grpcServerCallsSummary.WithLabelValues(
			pipeline, pluginName, procedure, string(rpcType), fromError(err).Code().String(),
		).Observe(float64(time.Since(begin)) / float64(time.Second))

		grpcServerCompletedTotal.WithLabelValues(
			pipeline, pluginName, procedure, string(rpcType),
		).Inc()

		return err
	}
}

func GrpcServerUnaryInterceptor(pipeline, pluginName string) grpc.UnaryServerInterceptor {
	grpcServerMetricsRegister.Do(func() {
		prometheus.MustRegister(grpcServerCallsSummary)
		prometheus.MustRegister(grpcServerCalledTotal)
		prometheus.MustRegister(grpcServerCompletedTotal)
		prometheus.MustRegister(grpcServerReceivedMsgTotal)
		prometheus.MustRegister(grpcServerSentMsgTotal)
	})

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		begin := time.Now()
		procedure := info.FullMethod
		rpcType := unary

		grpcServerCalledTotal.WithLabelValues(
			pipeline, pluginName, procedure, string(rpcType),
		).Inc()

		resp, err = handler(ctx, req)

		grpcServerCallsSummary.WithLabelValues(
			pipeline, pluginName, procedure, string(rpcType), fromError(err).Code().String(),
		).Observe(float64(time.Since(begin)) / float64(time.Second))

		grpcServerCompletedTotal.WithLabelValues(
			pipeline, pluginName, procedure, string(rpcType),
		).Inc()

		grpcServerReceivedMsgTotal.WithLabelValues(
			pipeline, pluginName, procedure, string(rpcType),
		).Inc()

		grpcServerSentMsgTotal.WithLabelValues(
			pipeline, pluginName, procedure, string(rpcType),
		).Inc()

		return resp, err
	}
}

func serverStreamType(info *grpc.StreamServerInfo) gRPCType {
	if info.IsClientStream && !info.IsServerStream {
		return clientStream
	} else if !info.IsClientStream && info.IsServerStream {
		return serverStream
	}
	return bidiStream
}

func fromError(err error) *status.Status {
	s, ok := status.FromError(err)
	// Mirror what the grpc server itself does, i.e. also convert context errors to status
	if !ok {
		s = status.FromContextError(err)
	}
	return s
}
