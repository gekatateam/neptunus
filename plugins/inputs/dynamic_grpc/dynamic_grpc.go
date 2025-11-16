package dynamicgrpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/jhump/protoreflect/v2/grpcdynamic"
	"kythe.io/kythe/go/util/datasize"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	dynamicgrpc "github.com/gekatateam/neptunus/plugins/common/dynamic_grpc"
	"github.com/gekatateam/neptunus/plugins/common/ider"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

const (
	modeServerSideStream = "ServerSideStream"
	modeAsServer         = "AsServer"
)

type DynamicGRPC struct {
	*core.BaseInput `mapstructure:"-"`
	Mode            string            `mapstructure:"mode"`
	ProtoFiles      []string          `mapstructure:"proto_files"`
	ImportPaths     []string          `mapstructure:"import_paths"`
	Procedures      []Procedure       `mapstructure:"procedures"`
	WaitForDelivery bool              `mapstructure:"wait_for_delivery"`
	LabelHeaders    map[string]string `mapstructure:"labelheaders"`
	*ider.Ider      `mapstructure:",squash"`

	// ServerSideStream
	Client      Client `mapstructure:"client"`
	clientConn  *grpc.ClientConn
	subscribers []*Subscriber

	// AsServer
	Server   Server `mapstructure:"server"`
	listener net.Listener
	server   *grpc.Server

	resolver   linker.Resolver
	cancelFunc context.CancelFunc
	doneCh     chan struct{}
}

type Procedure struct {
	Name           string            `mapstructure:"name"`
	InvokeRequest  string            `mapstructure:"invoke_request"`
	InvokeHeaders  map[string]string `mapstructure:"invoke_headers"`
	InvokeResponse string            `mapstructure:"invoke_response"`
}

type Client struct {
	dynamicgrpc.Client `mapstructure:",squash"`
	RetryAfter         time.Duration `mapstructure:"retry_after"`
}

type Server struct {
	dynamicgrpc.Server `mapstructure:",squash"`
}

func (i *DynamicGRPC) Init() error {
	if len(i.ProtoFiles) == 0 {
		return errors.New("at least one .proto file required")
	}

	if len(i.Procedures) == 0 {
		return errors.New("at least one procedure required")
	}

	i.Procedures = slices.CompactFunc(i.Procedures, func(a, b Procedure) bool {
		return a.Name == b.Name
	})

	if err := i.Ider.Init(); err != nil {
		return err
	}

	i.doneCh = make(chan struct{})

	compiler := &protocompile.Compiler{
		Resolver: protocompile.CompositeResolver{
			protocompile.WithStandardImports(&protocompile.SourceResolver{}),
			&protocompile.SourceResolver{ImportPaths: i.ImportPaths},
		},
	}

	f, err := compiler.Compile(context.Background(), i.ProtoFiles...)
	if err != nil {
		return fmt.Errorf("compilation error: %w", err)
	}

	i.resolver = f.AsResolver()

	switch i.Mode {
	case modeServerSideStream:
		if err := i.prepareClient(); err != nil {
			return err
		}
	case modeAsServer:
		if err := i.prepareServer(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown mode: %v", i.Mode)
	}

	return nil
}

func (i *DynamicGRPC) Close() error {
	switch i.Mode {
	case modeServerSideStream:
		i.cancelFunc()
		<-i.doneCh
		return i.clientConn.Close()
	case modeAsServer:
		i.server.GracefulStop()
		<-i.doneCh
		return i.listener.Close()
	default:
		// btw unreachable code, mode checks on init
		panic(fmt.Errorf("unknown mode: %v", i.Mode))
	}
}

func (i *DynamicGRPC) Run() {
	switch i.Mode {
	case modeServerSideStream:
		i.runServerSideStream()
	case modeAsServer:
		i.runAsServer()
	default:
		// btw unreachable code, mode checks on init
		panic(fmt.Errorf("unknown mode: %v", i.Mode))
	}

	close(i.doneCh)
}

func (i *DynamicGRPC) runAsServer() {
	i.Log.Info(fmt.Sprintf("starting grpc server on %v", i.Server.Address))
	if err := i.server.Serve(i.listener); err != nil {
		i.Log.Error("grpc server startup failed",
			"error", err.Error(),
		)
	} else {
		i.Log.Info("grpc server stopped")
	}
}

func (i *DynamicGRPC) runServerSideStream() {
	stub := grpcdynamic.NewStub(i.clientConn)
	ctx, cancel := context.WithCancel(context.Background())
	i.cancelFunc = cancel

	wg := &sync.WaitGroup{}
	for _, s := range i.subscribers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Subscribe(ctx, stub)
		}()
	}
	wg.Wait()
}

func (i *DynamicGRPC) prepareServer() error {
	if len(i.Server.Address) == 0 {
		return errors.New("address required")
	}

	listener, err := net.Listen("tcp", i.Server.Address)
	if err != nil {
		return fmt.Errorf("error creating listener: %v", err)
	}
	i.listener = listener

	services := make(map[protoreflect.FullName]*grpc.ServiceDesc)
	for _, rpc := range i.Procedures {
		if len(rpc.InvokeResponse) == 0 {
			return fmt.Errorf("%v: invoke response required", rpc.Name)
		}

		desc, err := i.resolver.FindDescriptorByName(protoreflect.FullName(rpc.Name))
		if err != nil {
			return fmt.Errorf("%v search failed: %w", rpc.Name, err)
		}

		if desc == nil {
			return fmt.Errorf("no such descriptor: %v", rpc.Name)
		}

		m, ok := desc.(protoreflect.MethodDescriptor)
		if !ok {
			return fmt.Errorf("%v is not a method", rpc.Name)
		}

		if m.IsStreamingServer() {
			return fmt.Errorf("%v is not an unary or client stream", rpc.Name)
		}

		respMsg := dynamicpb.NewMessage(m.Output())
		if err := protojson.Unmarshal([]byte(rpc.InvokeResponse), respMsg); err != nil {
			return fmt.Errorf("%v invoke response unmarshal failed: %w", rpc.Name, err)
		}

		service, ok := services[m.Parent().FullName()]
		if !ok {
			service = &grpc.ServiceDesc{ServiceName: string(m.Parent().FullName())}
			services[m.Parent().FullName()] = service
		}

		// create new handler per each RPC due to unique responses
		h := &Handler{
			BaseInput:       i.BaseInput,
			Ider:            i.Ider,
			LabelHeaders:    i.LabelHeaders,
			WaitForDelivery: i.WaitForDelivery,
			Procedure:       rpc.Name,
			RespMsg:         respMsg,
			RecvMsg:         m.Input(),
		}

		if m.IsStreamingClient() {
			service.Streams = append(service.Streams, grpc.StreamDesc{
				StreamName:    string(m.Name()),
				Handler:       h.HandleClientStream,
				ServerStreams: false,
				ClientStreams: true,
			})
		} else {
			service.Methods = append(service.Methods, grpc.MethodDesc{
				MethodName: string(m.Name()),
				Handler:    h.HandleUnary,
			})
		}
	}

	opts, err := i.Server.ServerOptions()
	if err != nil {
		return err
	}
	i.server = grpc.NewServer(opts...)

	for _, service := range services {
		i.server.RegisterService(service, nil)
	}

	return nil
}

func (i *DynamicGRPC) prepareClient() error {
	if len(i.Client.Address) == 0 {
		return errors.New("address required")
	}

	for _, rpc := range i.Procedures {
		if len(rpc.InvokeRequest) == 0 {
			return fmt.Errorf("%v: invoke request required", rpc.Name)
		}

		desc, err := i.resolver.FindDescriptorByName(protoreflect.FullName(rpc.Name))
		if err != nil {
			return fmt.Errorf("%v search failed: %w", rpc.Name, err)
		}

		if desc == nil {
			return fmt.Errorf("no such descriptor: %v", rpc.Name)
		}

		m, ok := desc.(protoreflect.MethodDescriptor)
		if !ok {
			return fmt.Errorf("%v is not a method", rpc.Name)
		}

		if m.IsStreamingClient() || !m.IsStreamingServer() {
			return fmt.Errorf("%v is not a server-side stream", rpc.Name)
		}

		initMsg := dynamicpb.NewMessage(m.Input())
		if err := protojson.Unmarshal([]byte(rpc.InvokeRequest), initMsg); err != nil {
			return fmt.Errorf("%v invoke request unmarshal failed: %w", rpc.Name, err)
		}

		initMD := make(metadata.MD)
		for k, v := range rpc.InvokeHeaders {
			initMD.Set(k, v)
		}

		i.subscribers = append(i.subscribers, &Subscriber{
			BaseInput:    i.BaseInput,
			Ider:         i.Ider,
			LabelHeaders: i.LabelHeaders,
			RetryAfter:   i.Client.RetryAfter,
			Procedure:    rpc.Name,
			method:       m,
			initMsg:      initMsg,
			initMD:       initMD,
		})
	}

	opts, err := i.Client.DialOptions()
	if err != nil {
		return err
	}

	conn, err := grpc.NewClient(i.Client.Address, opts...)
	if err != nil {
		return err
	}

	i.clientConn = conn
	return nil
}

func init() {
	plugins.AddInput("dynamic_grpc", func() core.Input {
		return &DynamicGRPC{
			Ider: &ider.Ider{},
			Client: Client{
				RetryAfter: 5 * time.Second,
				Client: dynamicgrpc.Client{
					TLSClientConfig: &tls.TLSClientConfig{},
				},
			},
			Server: Server{
				Server: dynamicgrpc.Server{
					MaxMessageSize:       4 * datasize.Mebibyte,
					NumStreamWorkers:     5,
					MaxConcurrentStreams: 5,
					TLSServerConfig:      &tls.TLSServerConfig{},
				},
			},
		}
	})
}
