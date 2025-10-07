package dynamicgrpc

import "google.golang.org/grpc/status"

func StatusFromError(err error) *status.Status {
	s, ok := status.FromError(err)
	// Mirror what the grpc server itself does, i.e. also convert context errors to status
	if !ok {
		s = status.FromContextError(err)
	}
	return s
}
