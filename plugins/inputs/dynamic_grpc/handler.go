package dynamicgrpc

import (
	"errors"
	"io"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/ider"
	"github.com/gekatateam/protomap"
	"github.com/gekatateam/protomap/interceptors"
)

type Handler struct {
	*core.BaseInput
	Procedure    string
	LabelHeaders map[string]string
	*ider.Ider

	RespMsg proto.Message
	RecvMsg protoreflect.MessageDescriptor
}

func (h *Handler) HandleClientStream(_ any, stream grpc.ServerStream) error {
	headers, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		headers = make(metadata.MD)
	}

	for {
		m := dynamicpb.NewMessage(h.RecvMsg)
		err := stream.RecvMsg(m)
		now := time.Now()

		if err != nil && errors.Is(err, io.EOF) {
			h.Log.Info("client stream closed")
			stream.SendMsg(h.RespMsg)
			return nil
		}

		if err != nil {
			h.Log.Error("message receiving error, stream aborted",
				"error", err,
			)
			h.Observe(metrics.EventFailed, time.Since(now))
			return nil
		}

		result, err := protomap.MessageToAny(m, interceptors.DurationDecoder, interceptors.TimeDecoder)
		if err != nil {
			h.Log.Error("message decoding error, message skipped",
				"error", err,
			)
			h.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		event := core.NewEventWithData(h.Procedure, result)
		for k, v := range h.LabelHeaders {
			if h := headers.Get(v); len(h) > 0 {
				event.SetLabel(k, strings.Join(h, "; "))
			}
		}

		h.Ider.Apply(event)
		h.Out <- event
		h.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func dialOptions(c Client) ([]grpc.DialOption, error) {
	var dialOpts []grpc.DialOption

	if len(c.Authority) > 0 {
		dialOpts = append(dialOpts, grpc.WithAuthority(c.Authority))
	}

	if len(c.UserAgent) > 0 {
		dialOpts = append(dialOpts, grpc.WithUserAgent(c.UserAgent))
	}

	dialOpts = append(dialOpts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                c.InactiveTransportPing,
		Timeout:             c.InactiveTransportAge,
		PermitWithoutStream: c.PermitWithoutStream,
	}))

	tlsConfig, err := c.TLSClientConfig.Config()
	if err != nil {
		return nil, err
	}

	if tlsConfig != nil {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	}

	return dialOpts, nil
}

func serverOptions(s Server) ([]grpc.ServerOption, error) {
	var serverOptions []grpc.ServerOption

	serverOptions = append(serverOptions,
		grpc.MaxRecvMsgSize(int(s.MaxMessageSize.Bytes())),
		grpc.MaxSendMsgSize(int(s.MaxMessageSize.Bytes())),
		grpc.NumStreamWorkers(s.NumStreamWorkers),
		grpc.MaxConcurrentStreams(s.MaxConcurrentStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     s.MaxConnectionIdle,
			MaxConnectionAge:      s.MaxConnectionAge,
			MaxConnectionAgeGrace: s.MaxConnectionGrace,
			Time:                  s.InactiveTransportPing,
			Timeout:               s.InactiveTransportAge,
		}),
	)

	tlsConfig, err := s.TLSServerConfig.Config()
	if err != nil {
		return nil, err
	}

	if tlsConfig != nil {
		serverOptions = append(serverOptions, grpc.Creds(credentials.NewTLS(tlsConfig)))
	} else {
		serverOptions = append(serverOptions, grpc.Creds(insecure.NewCredentials()))

	}

	return serverOptions, nil
}
