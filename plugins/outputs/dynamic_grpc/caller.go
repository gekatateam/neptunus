package dynamicgrpc

import (
	"github.com/jhump/protoreflect/v2/grpcdynamic"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
)

type Caller struct {
	*core.BaseOutput
	*batcher.Batcher[*core.Event]
	*retryer.Retryer

	labelHeaders map[string]string
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
		c.sendFunc(buf)
	})
}

func (c *Caller) sendUnary(buf []*core.Event) {}

func (c *Caller) sendBulk(buf []*core.Event) {}
