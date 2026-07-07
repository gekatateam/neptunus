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
	"net"
	"regexp"
	"sync"
	"time"

	"github.com/jhump/protoreflect/v2/grpcdynamic"
	"kythe.io/kythe/go/util/datasize"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/slices"
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
	modeAsServer = "AsServer"
)

const (
	behaviourBroadcast = "Broadcast"
	behaviourRandom    = "Random"
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
	headers      metadata.MD
	callersPool  *pool.Pool[*core.Event, string]
	successCodes map[codes.Code]struct{}

	// AsServer
	Server   Server `mapstructure:"server"`
	listener net.Listener
	server   *grpc.Server
	router   *router
	rpcs     map[protoreflect.FullName]protoreflect.MessageDescriptor

	resolver linker.Resolver
}

type Client struct {
	SuccessCodes                  []int32        `mapstructure:"success_codes"`
	SuccessMessage                *regexp.Regexp `mapstructure:"success_message"`
	IdleTimeout                   time.Duration  `mapstructure:"idle_timeout"`
	InvokeTimeout                 time.Duration  `mapstructure:"invoke_timeout"`
	dynamicgrpc.Client            `mapstructure:",squash"`
	*batcher.Batcher[*core.Event] `mapstructure:",squash"`
	*retryer.Retryer              `mapstructure:",squash"`
}

type Server struct {
	Behaviour string `mapstructure:"behaviour"`
	// WaitForSubscribers bool     `mapstructure:"wait_for_subscribers"`
	Procedures         []string `mapstructure:"procedures"`
	dynamicgrpc.Server `mapstructure:",squash"`
	*retryer.Retryer   `mapstructure:",squash"`
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
		if err := o.prepareClient(); err != nil {
			return err
		}
	case modeAsServer:
		if err := o.prepareServer(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown mode: %v", o.Mode)
	}

	return nil
}

func (o *DynamicGRPC) Run() {
	switch o.Mode {
	case modeAsClient:
		o.runAsClient()
	case modeAsServer:
		o.runAsServer()
	default:
		panic(fmt.Errorf("unknown mode: %v", o.Mode))
	}
}

func (o *DynamicGRPC) Close() error {
	switch o.Mode {
	case modeAsClient:
		return o.clientConn.Close()
	case modeAsServer:
		return o.listener.Close()
	default:
		panic(fmt.Errorf("unknown mode: %v", o.Mode))
	}
}

func (o *DynamicGRPC) prepareClient() error {
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
			o.Log.Warn("it is strongly not recommended to use Unknown (2) code as a successful one")
		}

		o.successCodes[codes.Code(code)] = struct{}{}
	}

	if o.Client.IdleTimeout > 0 && o.Client.IdleTimeout < time.Minute {
		o.Client.IdleTimeout = time.Minute
	}

	o.clientConn = conn
	o.callersPool = pool.New(o.newCaller)

	return nil
}

func (o *DynamicGRPC) prepareServer() error {
	if len(o.Server.Address) == 0 {
		return errors.New("address required")
	}

	if len(o.Server.Procedures) == 0 {
		return errors.New("at least one procedure required")
	}

	switch o.Server.Behaviour {
	case behaviourBroadcast, behaviourRandom:
	default:
		return fmt.Errorf("unknown behaviour: %v", o.Server.Behaviour)
	}

	listener, err := net.Listen("tcp", o.Server.Address)
	if err != nil {
		return fmt.Errorf("error creating listener: %w", err)
	}
	o.listener = listener

	o.router = &router{
		mu:               &sync.Mutex{},
		wg:               &sync.WaitGroup{},
		pubs:             make(map[string]*publisher, len(o.Server.Procedures)),
		closed:           false,
		newPublisherFunc: o.newPublisher,
	}
	rpcs := make(map[protoreflect.FullName]protoreflect.MessageDescriptor)

	services := make(map[protoreflect.FullName]*grpc.ServiceDesc)
	for _, rpc := range slices.Unique(o.Server.Procedures) {
		desc, err := o.resolver.FindDescriptorByName(protoreflect.FullName(rpc))
		if err != nil {
			return fmt.Errorf("%v search failed: %w", rpc, err)
		}

		if desc == nil {
			return fmt.Errorf("no such descriptor: %v", rpc)
		}

		m, ok := desc.(protoreflect.MethodDescriptor)
		if !ok {
			return fmt.Errorf("%v is not a method", rpc)
		}

		if !(m.IsStreamingServer() && !m.IsStreamingClient()) {
			return fmt.Errorf("%v is not a server stream", rpc)
		}

		rpcs[m.FullName()] = m.Output()

		service, ok := services[m.Parent().FullName()]
		if !ok {
			service = &grpc.ServiceDesc{ServiceName: string(m.Parent().FullName())}
			services[m.Parent().FullName()] = service
		}

		s := &Streamer{
			BaseOutput:      o.BaseOutput,
			procedure:       rpc,
			subscribeFunc:   o.router.subscribe,
			unsubscribeFunc: o.router.unsubscribe,
			recvMsg:         m.Input(),
		}

		service.Streams = append(service.Streams, grpc.StreamDesc{
			StreamName:    string(m.Name()),
			Handler:       s.HandleServerStream,
			ServerStreams: true,
			ClientStreams: false,
		})
	}

	opts, err := o.Server.ServerOptions()
	if err != nil {
		return err
	}
	o.server = grpc.NewServer(opts...)

	for _, service := range services {
		o.server.RegisterService(service, nil)
	}

	o.rpcs = rpcs
	return nil
}

func (o *DynamicGRPC) runAsClient() {
	clearTicker := time.NewTicker(time.Minute)
	if o.Client.IdleTimeout == 0 {
		clearTicker.Stop()
	}

	defer o.callersPool.Close()

MAIN_LOOP:
	for {
		select {
		case e, ok := <-o.In:
			if !ok {
				clearTicker.Stop()
				break MAIN_LOOP
			}

			if (o.callersPool.LastWrite(e.RoutingKey) != time.Time{}) {
				goto DESCRIPTOR_FOUND_AND_CORRECT
			}

			if _, err := o.descriptorForClient(protoreflect.FullName(e.RoutingKey)); err != nil {
				o.Log.Error("event processing failed",
					"error", err,
					elog.EventGroup(e),
				)
				o.Observe(metrics.EventFailed, 0)
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
}

func (o *DynamicGRPC) runAsServer() {
	wg := o.router.wg

	wg.Go(func() {
		o.Log.Info(fmt.Sprintf("starting grpc server on %v", o.Server.Address))
		if err := o.server.Serve(o.listener); err != nil {
			o.Log.Error("grpc server startup failed",
				"error", err,
			)
		} else {
			o.Log.Info("grpc server stopped")
		}
	})

	for e := range o.In {
		if _, ok := o.rpcs[protoreflect.FullName(e.RoutingKey)]; !ok {
			o.Log.Error("event processing failed",
				"error", fmt.Errorf("no such method: %v", e.RoutingKey),
				elog.EventGroup(e),
			)
			o.Observe(metrics.EventFailed, 0)
			o.Done <- e
			continue
		}

		o.router.push(e)
	}

	o.router.close()

	wg.Go(func() {
		o.Log.Info("stopping grpc server")
		o.server.GracefulStop()
	})

	o.router.stop()

	wg.Wait()
}

func (o *DynamicGRPC) newCaller(rpc string) pool.Runner[*core.Event] {
	m, _ := o.descriptorForClient(protoreflect.FullName(rpc))

	c := &Caller{
		BaseOutput:   o.BaseOutput,
		Batcher:      o.Client.Batcher,
		Retryer:      o.Client.Retryer,
		headerLabels: o.HeaderLabels,
		timeout:      o.Client.InvokeTimeout,
		successMsg:   o.Client.SuccessMessage,
		successCodes: o.successCodes,
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

func (o *DynamicGRPC) newPublisher(rpc string) *publisher {
	return &publisher{
		BaseOutput:     o.BaseOutput,
		behavior:       o.Server.Behaviour,
		rpc:            rpc,
		waitForSubs:    false, // o.Server.WaitForSubscribers,
		respMsg:        o.rpcs[protoreflect.FullName(rpc)],
		subs:           make([]subscription, 0),
		stop:           make(chan struct{}),
		events:         make(chan *core.Event, 1),
		subscription:   make(chan subscription, 1),
		unsubscription: make(chan subscription, 1),
	}
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
	p := func() core.Output {
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
			Server: Server{
				Behaviour: behaviourRandom,
				// WaitForSubscribers: true,
				Server: dynamicgrpc.Server{
					MaxMessageSize:       4 * datasize.Mebibyte,
					NumStreamWorkers:     5,
					MaxConcurrentStreams: 5,
					TLSServerConfig:      &tls.TLSServerConfig{},
				},
				Retryer: &retryer.Retryer{
					RetryAttempts: 0,
					RetryAfter:    5 * time.Second,
				},
			},
		}
	}

	plugins.AddOutput("dynamic_grpc", func() core.Output {
		return p()
	})

	plugins.AddOutput("grpc", func() core.Output {
		return p()
	})
}
