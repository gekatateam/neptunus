package dynamicgrpc

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/jhump/protoreflect/v2/grpcdynamic"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/gekatateam/protomap"
	"github.com/gekatateam/protomap/interceptors"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	dynamicgrpc "github.com/gekatateam/neptunus/plugins/common/dynamic_grpc"
	"github.com/gekatateam/neptunus/plugins/common/elog"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
)

type Caller struct {
	*core.BaseOutput
	*batcher.Batcher[*core.Event]
	*retryer.Retryer

	headerLabels map[string]string
	successCodes map[codes.Code]struct{}
	successMsg   *regexp.Regexp
	timeout      time.Duration
	method       protoreflect.MethodDescriptor
	stub         *grpcdynamic.Stub
	sendFunc     func(buf []*core.Event)

	input chan *core.Event
}

func (c *Caller) Push(e *core.Event) {
	c.input <- e
}
func (c *Caller) Close() error {
	close(c.input)
	return nil
}

func (c *Caller) Run() {
	c.Log.Info(fmt.Sprintf("caller for %v spawned", c.method.FullName()))

	c.Batcher.Run(c.input, func(buf []*core.Event) {
		if len(buf) == 0 {
			return
		}

		c.sendFunc(buf)
	})

	c.Log.Info(fmt.Sprintf("caller for %v closed", c.method.FullName()))
}

func (c *Caller) sendUnary(buf []*core.Event) {
	for _, e := range buf {
		now := time.Now()

		msg := dynamicpb.NewMessage(c.method.Input())
		if err := protomap.AnyToMessage(e.Data, msg, interceptors.DurationEncoder, interceptors.TimeEncoder); err != nil {
			c.Log.Error("event processing failed",
				"error", fmt.Errorf("encoding failed: %w", err),
				elog.EventGroup(e),
			)
			c.Observe(metrics.EventFailed, time.Since(now))
			c.Done <- e
			continue
		}

		headers := make(metadata.MD)
		for k, v := range c.headerLabels {
			if val, ok := e.GetLabel(v); ok {
				headers.Set(k, val)
			}
		}
		ctx := metadata.NewOutgoingContext(context.Background(), headers)

		err := c.Retryer.Do("execute unary rpc", c.Log, func() error {
			invokeCtx, cancel := context.WithTimeout(ctx, c.timeout)
			defer cancel()

			resp, invokeErr := c.stub.InvokeRpc(invokeCtx, c.method, msg)
			status := dynamicgrpc.StatusFromError(invokeErr)

			jsonResp, err := protojson.Marshal(resp)
			if err != nil {
				c.Log.Warn("rpc response body marshal failed",
					"error", err,
					elog.EventGroup(e),
				)
			}

			if _, ok := c.successCodes[status.Code()]; ok {
				c.Log.Debug(fmt.Sprintf("rpc invoked successfully with code: %v; message: %v; response: %v", status.Code(), status.Message(), string(jsonResp)),
					elog.EventGroup(e),
				)
				return nil
			} else {
				if c.successMsg != nil && c.successMsg.MatchString(status.Message()) {
					c.Log.Debug(fmt.Sprintf("rpc invoked with code: %v; message: %v; response: %v; "+
						"but RPC status message matches configured regexp", status.Code(), status.Message(), string(jsonResp)),
						elog.EventGroup(e),
					)
					return nil
				} else {
					return fmt.Errorf("rpc invoke failed with code: %v; message: %v; response: %v", status.Code(), status.Message(), string(jsonResp))
				}
			}
		})

		c.Done <- e
		if err != nil {
			c.Log.Error("event processing failed",
				"error", err,
				elog.EventGroup(e),
			)
			c.Observe(metrics.EventFailed, time.Since(now))
		} else {
			c.Log.Debug("event processed",
				elog.EventGroup(e),
			)
			c.Observe(metrics.EventAccepted, time.Since(now))
		}
	}
}

func (c *Caller) sendBulk(buf []*core.Event) {
	headers := make(metadata.MD)
	for k, v := range c.headerLabels {
		if val, ok := buf[0].GetLabel(v); ok {
			headers.Set(k, val)
		}
	}
	ctx := metadata.NewOutgoingContext(context.Background(), headers)

	var stream *grpcdynamic.ClientStream
	for _, e := range buf {
		now := time.Now()

		msg := dynamicpb.NewMessage(c.method.Input())
		if err := protomap.AnyToMessage(e.Data, msg, interceptors.DurationEncoder, interceptors.TimeEncoder); err != nil {
			c.Log.Error("event processing failed",
				"error", fmt.Errorf("encoding failed: %w", err),
				elog.EventGroup(e),
			)
			c.Done <- e
			c.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

	BEFORE_STREAM_CHECK:
		if stream == nil {
			s, err := c.clientStream(ctx)
			if err != nil {
				c.Log.Error("event processing failed",
					"error", err,
					elog.EventGroup(e),
				)
				c.Done <- e
				c.Observe(metrics.EventFailed, time.Since(now))
				continue
			}

			stream = s
		}

		if err := stream.SendMsg(msg); err != nil {
			c.Log.Error("send to stream failed",
				"error", err,
				elog.EventGroup(e),
			)

			// if SendMsg failed, stream is already aborted
			stream = nil
			goto BEFORE_STREAM_CHECK
		}

		c.Log.Debug("message sent to stream",
			elog.EventGroup(e),
		)
		c.Done <- e
		c.Observe(metrics.EventAccepted, time.Since(now))
	}

	if stream == nil {
		c.Log.Error(fmt.Sprintf("stream %v already aborted, can't receive response message", c.method.FullName()))
		return
	}

	resp, err := stream.CloseAndReceive()
	if err != nil {
		c.Log.Error(fmt.Sprintf("stream %v closing failed", c.method.FullName()),
			"error", err,
		)
		return
	}

	jsonResp, err := protojson.Marshal(resp)
	if err != nil {
		c.Log.Warn(fmt.Sprintf("stream %v rpc response body marshal failed", c.method.FullName()),
			"error", err,
		)
		return
	}

	c.Log.Debug(fmt.Sprintf("stream %v completed with response: %v", c.method.FullName(), string(jsonResp)))
}

func (c *Caller) clientStream(ctx context.Context) (*grpcdynamic.ClientStream, error) {
	var stream *grpcdynamic.ClientStream

	err := c.Retryer.Do(fmt.Sprintf("create %v stream", c.method.FullName()), c.Log, func() error {
		s, err := c.stub.InvokeRpcClientStream(ctx, c.method)
		if err != nil {
			return err
		}

		stream = s
		return nil
	})

	return stream, err
}
