package dynamicgrpc

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/gekatateam/protomap"
	"github.com/gekatateam/protomap/interceptors"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/ider"
)

type Handler struct {
	*core.BaseInput
	*ider.Ider
	Procedure    string
	LabelHeaders map[string]string

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
			h.Log.Info("client stream closed",
				"procedure", h.Procedure,
			)
			stream.SendMsg(h.RespMsg)
			return nil
		}

		if err != nil {
			h.Log.Error("message receiving error, stream aborted",
				"error", err,
				"procedure", h.Procedure,
			)
			h.Observe(metrics.EventFailed, time.Since(now))
			return nil
		}

		result, err := protomap.MessageToAny(m, interceptors.DurationDecoder, interceptors.TimeDecoder)
		if err != nil {
			h.Log.Error("message decoding error, message skipped",
				"error", err,
				"procedure", h.Procedure,
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

func (h *Handler) HandleUnary(_ any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	now := time.Now()

	m := dynamicpb.NewMessage(h.RecvMsg)
	if err := dec(m); err != nil {
		h.Log.Error("message decoding error",
			"error", err,
			"procedure", h.Procedure,
		)
		h.Observe(metrics.EventFailed, time.Since(now))
		return nil, err
	}

	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		headers = make(metadata.MD)
	}

	result, err := protomap.MessageToAny(m, interceptors.DurationDecoder, interceptors.TimeDecoder)
	if err != nil {
		h.Log.Error("message decoding error",
			"error", err,
			"procedure", h.Procedure,
		)
		h.Observe(metrics.EventFailed, time.Since(now))
		return nil, err
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

	return h.RespMsg, nil
}
