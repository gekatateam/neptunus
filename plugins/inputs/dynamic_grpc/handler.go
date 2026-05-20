package dynamicgrpc

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

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
	procedure       string
	labelHeaders    map[string]string
	waitForDelivery bool

	respMsg proto.Message
	recvMsg protoreflect.MessageDescriptor
}

func (h *Handler) HandleClientStream(_ any, stream grpc.ServerStream) error {
	headers, _ := metadata.FromIncomingContext(stream.Context())
	peer, _ := peer.FromContext(stream.Context())

	wg := &sync.WaitGroup{}
	for {
		m := dynamicpb.NewMessage(h.recvMsg)
		err := stream.RecvMsg(m)
		now := time.Now()

		if err != nil && errors.Is(err, io.EOF) {
			h.Log.Info("client stream closed",
				"procedure", h.procedure,
				"peer", peer.String(),
			)
			wg.Wait()
			stream.SendMsg(h.respMsg)
			return nil
		}

		if err != nil {
			h.Log.Error("message receiving error, stream aborted",
				"error", err,
				"procedure", h.procedure,
				"peer", peer.String(),
			)
			h.Observe(metrics.EventFailed, time.Since(now))
			wg.Wait()
			return nil
		}

		result, err := protomap.MessageToAny(m, interceptors.DurationDecoder, interceptors.TimeDecoder)
		if err != nil {
			h.Log.Error("message decoding error, message skipped",
				"error", err,
				"procedure", h.procedure,
				"peer", peer.String(),
			)
			h.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		event := core.NewEventWithData(h.procedure, result)
		event.SetLabel("peer", peer.String())
		for k, v := range h.labelHeaders {
			if h := headers.Get(v); len(h) > 0 {
				event.SetLabel(k, strings.Join(h, "; "))
			}
		}

		if h.waitForDelivery {
			wg.Add(1)
			event.AddHook(wg.Done)
		}

		h.Ider.Apply(event)
		h.Out <- event
		h.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func (h *Handler) HandleUnary(_ any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	now := time.Now()
	wg := &sync.WaitGroup{}

	headers, _ := metadata.FromIncomingContext(ctx)
	peer, _ := peer.FromContext(ctx)

	m := dynamicpb.NewMessage(h.recvMsg)
	if err := dec(m); err != nil {
		h.Log.Error("message decoding error",
			"error", err,
			"procedure", h.procedure,
			"peer", peer.String(),
		)
		h.Observe(metrics.EventFailed, time.Since(now))
		return nil, err
	}

	result, err := protomap.MessageToAny(m, interceptors.DurationDecoder, interceptors.TimeDecoder)
	if err != nil {
		h.Log.Error("message decoding error",
			"error", err,
			"procedure", h.procedure,
			"peer", peer.String(),
		)
		h.Observe(metrics.EventFailed, time.Since(now))
		return nil, err
	}

	event := core.NewEventWithData(h.procedure, result)
	event.SetLabel("peer", peer.String())
	for k, v := range h.labelHeaders {
		if h := headers.Get(v); len(h) > 0 {
			event.SetLabel(k, strings.Join(h, "; "))
		}
	}

	if h.waitForDelivery {
		wg.Add(1)
		event.AddHook(wg.Done)
	}

	h.Ider.Apply(event)
	h.Out <- event
	h.Observe(metrics.EventAccepted, time.Since(now))

	wg.Wait()
	return h.respMsg, nil
}
