package metrics

import "google.golang.org/grpc/status"

type gRPCType string

const (
	unary        gRPCType = "unary"
	serverStream gRPCType = "server_stream"
	clientStream gRPCType = "client_stream"
	bidiStream   gRPCType = "bidi_stream"
)

func fromGrpcError(err error) *status.Status {
	s, ok := status.FromError(err)
	// Mirror what the grpc server itself does, i.e. also convert context errors to status
	if !ok {
		s = status.FromContextError(err)
	}
	return s
}
