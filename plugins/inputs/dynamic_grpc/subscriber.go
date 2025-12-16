package dynamicgrpc

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/jhump/protoreflect/v2/grpcdynamic"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/gekatateam/protomap"
	"github.com/gekatateam/protomap/interceptors"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/ider"
)

type Subscriber struct {
	*core.BaseInput
	*ider.Ider
	Procedure    string
	LabelHeaders map[string]string
	RetryAfter   time.Duration

	method  protoreflect.MethodDescriptor
	initMsg proto.Message
	initMD  metadata.MD
}

func (s *Subscriber) Subscribe(initCtx context.Context, stub *grpcdynamic.Stub) {
	ctx := metadata.NewOutgoingContext(initCtx, s.initMD)

	var ss *grpcdynamic.ServerStream
STREAM_INVOKE_LOOP:
	for {
		select {
		case <-ctx.Done():
			s.Log.Info("server-side stream context canceled",
				"procedure", s.Procedure,
			)
			return
		default:
			stream, err := stub.InvokeRpcServerStream(ctx, s.method, s.initMsg)
			if err != nil {
				s.Log.Error("invoke server-side stream failed",
					"error", err,
					"procedure", s.Procedure,
				)
				time.Sleep(s.RetryAfter)
			} else {
				s.Log.Info("server-side stream created",
					"procedure", s.Procedure,
				)
				ss = stream
				break STREAM_INVOKE_LOOP
			}
		}
	}

	header, err := ss.Header()
	if err != nil {
		s.Log.Error("receive headers failed",
			"error", err,
			"procedure", s.Procedure,
		)
		time.Sleep(s.RetryAfter)
		goto STREAM_INVOKE_LOOP
	}

	var now time.Time
STREAM_READ_LOOP:
	for {
		select {
		case <-ctx.Done():
			s.Log.Info("server-side stream context canceled",
				"procedure", s.Procedure,
			)
			return
		default:
			msg, err := ss.RecvMsg()
			now = time.Now()
			if err != nil {
				// io.EOF means stream is closed gracefuly
				if errors.Is(err, io.EOF) {
					goto STREAM_INVOKE_LOOP
				}

				// fires on shutdown
				if status.Code(err) == codes.Canceled {
					goto STREAM_INVOKE_LOOP
				}

				// any other error means that the stream is aborted and the error contains the RPC status
				s.Log.Error("receive from server-side stream failed",
					"error", err,
					"procedure", s.Procedure,
				)
				s.Observe(metrics.EventFailed, time.Since(now))
				time.Sleep(s.RetryAfter)
				goto STREAM_INVOKE_LOOP
			}

			result, err := protomap.MessageToAny(msg.ProtoReflect(), interceptors.DurationDecoder, interceptors.TimeDecoder)
			if err != nil {
				s.Log.Error("message decoding failed",
					"error", err,
					"procedure", s.Procedure,
				)
				s.Observe(metrics.EventFailed, time.Since(now))
				continue STREAM_READ_LOOP
			}

			event := core.NewEventWithData(string(s.method.FullName()), result)
			for k, v := range s.LabelHeaders {
				if h := header.Get(v); len(h) > 0 {
					event.SetLabel(k, strings.Join(h, "; "))
				}
			}

			s.Ider.Apply(event)
			s.Out <- event
			s.Observe(metrics.EventAccepted, time.Since(now))
		}
	}
}
