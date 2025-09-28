package dynamicgrpc

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/jhump/protoreflect/v2/grpcdynamic"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/gekatateam/protomap"
	"github.com/gekatateam/protomap/interceptors"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
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
	c.Batcher.Run(c.input, func(buf []*core.Event) {
		if len(buf) == 0 {
			return
		}

		c.sendFunc(buf)
	})
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
			status, ok := status.FromError(invokeErr)
			if !ok { // if error does not contains a status, return it as is
				return invokeErr
			}

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

func (c *Caller) sendBulk(buf []*core.Event) {}
