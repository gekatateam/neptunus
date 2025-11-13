package dynamicgrpc

// as CLIENT:
// - if unary - send every event
// - if client stream - send all events in batch
//
// as SERVER
// - if server stream - send every event to all connected clients

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jhump/protoreflect/v2/grpcdynamic"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	dynamicgrpc "github.com/gekatateam/neptunus/plugins/common/dynamic_grpc"
	"github.com/gekatateam/neptunus/plugins/common/elog"
	"github.com/gekatateam/neptunus/plugins/common/pool"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

const (
	modeAsClient = "AsClient"
)

type DynamicGRPC struct {
	*core.BaseOutput `mapstructure:"-"`
	Mode             string            `mapstructure:"mode"`
	ProtoFiles       []string          `mapstructure:"proto_files"`
	ImportPaths      []string          `mapstructure:"import_paths"`
	Headers          map[string]string `mapstructure:"headers"`
	HeaderLabels     map[string]string `mapstructure:"headerlabels"`

	// AsClient
	Client       Client `mapstructure:"client"`
	clientConn   *grpc.ClientConn
	successCodes map[codes.Code]struct{}
	successMsg   *regexp.Regexp

	headers     metadata.MD
	callersPool *pool.Pool[*core.Event, string]
	resolver    linker.Resolver
}

type Client struct {
	SuccessCodes                  []int32       `mapstructure:"success_codes"`
	SuccessMessage                string        `mapstructure:"success_message"`
	IdleTimeout                   time.Duration `mapstructure:"idle_timeout"`
	InvokeTimeout                 time.Duration `mapstructure:"invoke_timeout"`
	dynamicgrpc.Client            `mapstructure:",squash"`
	*batcher.Batcher[*core.Event] `mapstructure:",squash"`
	*retryer.Retryer              `mapstructure:",squash"`
}

func (o *DynamicGRPC) Init() error {
	if len(o.ProtoFiles) == 0 {
		return errors.New("at least one .proto file required")
	}

	compiler := &protocompile.Compiler{
		Resolver: protocompile.CompositeResolver{
			protocompile.WithStandardImports(&protocompile.SourceResolver{}),
			&protocompile.SourceResolver{ImportPaths: o.ImportPaths},
		},
	}

	f, err := compiler.Compile(context.Background(), o.ProtoFiles...)
	if err != nil {
		return fmt.Errorf("compilation error: %w", err)
	}

	o.resolver = f.AsResolver()

	o.headers = make(metadata.MD, len(o.Headers))
	for k, v := range o.Headers {
		o.headers.Set(k, v)
	}

	switch o.Mode {
	case modeAsClient:
		if len(o.Client.Address) == 0 {
			return errors.New("address required")
		}

		options, err := o.Client.DialOptions()
		if err != nil {
			return err
		}

		conn, err := grpc.NewClient(o.Client.Address, options...)
		if err != nil {
			return err
		}

		o.successCodes = make(map[codes.Code]struct{})
		for _, code := range o.Client.SuccessCodes {
			if code == int32(codes.Unknown) {
				o.Log.Warn("it is strongly recommended not to use Unknown (2) code as a successful one")
			}

			o.successCodes[codes.Code(code)] = struct{}{}
		}

		o.successMsg = nil
		if len(o.Client.SuccessMessage) > 0 {
			r, err := regexp.Compile(o.Client.SuccessMessage)
			if err != nil {
				return err
			}

			o.successMsg = r
		}

		if o.Client.IdleTimeout > 0 && o.Client.IdleTimeout < time.Minute {
			o.Client.IdleTimeout = time.Minute
		}

		o.clientConn = conn
		o.callersPool = pool.New(o.newCaller)
	default:
		return fmt.Errorf("unknown mode: %v", o.Mode)
	}

	return nil
}

func (o *DynamicGRPC) Run() {
	clearTicker := time.NewTicker(time.Minute)
	if o.Client.IdleTimeout == 0 {
		clearTicker.Stop()
	}

MAIN_LOOP:
	for {
		select {
		case e, ok := <-o.In:
			if !ok {
				clearTicker.Stop()
				break MAIN_LOOP
			}

			now := time.Now()
			if (o.callersPool.LastWrite(e.RoutingKey) != time.Time{}) {
				goto DESCRIPTOR_FOUND_AND_CORRECT
			}

			if _, err := o.descriptorForClient(protoreflect.FullName(e.RoutingKey)); err != nil {
				o.Log.Error("event processing failed",
					"error", err,
					elog.EventGroup(e),
				)
				o.Observe(metrics.EventFailed, time.Since(now))
				o.Done <- e
				continue
			}

		DESCRIPTOR_FOUND_AND_CORRECT:
			o.callersPool.Get(e.RoutingKey).Push(e)
		case <-clearTicker.C:
			for _, pipeline := range o.callersPool.Keys() {
				if time.Since(o.callersPool.LastWrite(pipeline)) > o.Client.IdleTimeout {
					o.callersPool.Remove(pipeline)
				}
			}
		}
	}

	o.callersPool.Close()
}

func (o *DynamicGRPC) Close() error {
	return o.clientConn.Close()
}

func (o *DynamicGRPC) newCaller(rpc string) pool.Runner[*core.Event] {
	m, _ := o.descriptorForClient(protoreflect.FullName(rpc))

	c := &Caller{
		BaseOutput:   o.BaseOutput,
		Batcher:      o.Client.Batcher,
		Retryer:      o.Client.Retryer,
		headerLabels: o.HeaderLabels,
		timeout:      o.Client.InvokeTimeout,
		successCodes: o.successCodes,
		successMsg:   o.successMsg,
		headers:      o.headers,
		method:       m,
		stub:         grpcdynamic.NewStub(o.clientConn),
		input:        make(chan *core.Event),
	}

	if m.IsStreamingClient() {
		c.sendFunc = c.sendBulk
	} else {
		c.sendFunc = c.sendUnary
	}

	return c
}

func (o *DynamicGRPC) descriptorForClient(name protoreflect.FullName) (protoreflect.MethodDescriptor, error) {
	desc, err := o.resolver.FindDescriptorByName(name)
	if err != nil {
		return nil, fmt.Errorf("%v search failed: %w", name, err)
	}

	if desc == nil {
		return nil, fmt.Errorf("no such descriptor: %v", name)
	}

	m, ok := desc.(protoreflect.MethodDescriptor)
	if !ok {
		return nil, fmt.Errorf("%v is not a method", name)
	}

	if m.IsStreamingServer() {
		return nil, fmt.Errorf("%v is not an unary or client stream", name)
	}

	return m, nil
}

func init() {
	plugins.AddOutput("dynamic_grpc", func() core.Output {
		return &DynamicGRPC{
			Client: Client{
				SuccessCodes:  []int32{0},
				IdleTimeout:   time.Hour,
				InvokeTimeout: 30 * time.Second,
				Client: dynamicgrpc.Client{
					TLSClientConfig: &tls.TLSClientConfig{},
				},
				Batcher: &batcher.Batcher[*core.Event]{
					Buffer:   100,
					Interval: 5 * time.Second,
				},
				Retryer: &retryer.Retryer{
					RetryAttempts: 0,
					RetryAfter:    5 * time.Second,
				},
			},
		}
	})
}
