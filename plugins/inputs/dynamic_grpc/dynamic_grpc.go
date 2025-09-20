package dynamicgrpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/jhump/protoreflect/v2/grpcdynamic"
	"kythe.io/kythe/go/util/datasize"

	"github.com/bufbuild/protocompile"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	dynamicgrpc "github.com/gekatateam/neptunus/plugins/common/dynamic_grpc"
	"github.com/gekatateam/neptunus/plugins/common/ider"
	"github.com/gekatateam/neptunus/plugins/common/tls"
	"github.com/gekatateam/protomap"
	"github.com/gekatateam/protomap/interceptors"
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
	Procedure       string            `mapstructure:"procedure"`
	LabelHeaders    map[string]string `mapstructure:"labelheaders"`
	*ider.Ider      `mapstructure:",squash"`

	// ServerSideStream
	Client     dynamicgrpc.Client `mapstructure:"client"`
	clientConn *grpc.ClientConn
	initMsg    proto.Message
	initMD     metadata.MD

	// AsServer
	Server   dynamicgrpc.Server `mapstructure:"server"`
	listener net.Listener
	server   *grpc.Server
	respMsg  proto.Message

	method     protoreflect.MethodDescriptor
	cancelFunc context.CancelFunc
	doneCh     chan struct{}
}

func (i *DynamicGRPC) Init() error {
	if len(i.ProtoFiles) == 0 {
		return errors.New("at least one .proto file required")
	}

	if len(i.Procedure) == 0 {
		return errors.New("procedure name required")
	}

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

	r := f.AsResolver()
	desc, err := r.FindDescriptorByName(protoreflect.FullName(i.Procedure))
	if err != nil {
		return fmt.Errorf("%v search failed: %w", i.Procedure, err)
	}

	if desc == nil {
		return fmt.Errorf("no such descriptor: %v", i.Procedure)
	}

	m, ok := desc.(protoreflect.MethodDescriptor)
	if !ok {
		return fmt.Errorf("%v is not a method", i.Procedure)
	}

	i.method = m

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
	ctx, cancel := context.WithCancel(metadata.NewOutgoingContext(context.Background(), i.initMD))

	i.cancelFunc = cancel
	var ss *grpcdynamic.ServerStream

STREAM_INVOKE_LOOP:
	for {
		select {
		case <-ctx.Done():
			i.Log.Info("server-side stream context canceled")
			return
		default:
			stream, err := stub.InvokeRpcServerStream(ctx, i.method, i.initMsg)
			if err != nil {
				i.Log.Error("invoke server-side stream failed",
					"error", err,
				)
				time.Sleep(i.Client.RetryAfter)
			} else {
				i.Log.Info("server-side stream created")
				ss = stream
				break STREAM_INVOKE_LOOP
			}
		}
	}

	header, err := ss.Header()
	if err != nil {
		i.Log.Error("receive headers failed",
			"error", err,
		)
		time.Sleep(i.Client.RetryAfter)
		goto STREAM_INVOKE_LOOP
	}

STREAM_READ_LOOP:
	for {
		select {
		case <-ctx.Done():
			i.Log.Info("server-side stream context canceled")
			return
		default:
			msg, err := ss.RecvMsg()
			now := time.Now()
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
				i.Log.Error("receive from server-side stream failed",
					"error", err,
				)
				i.Observe(metrics.EventFailed, time.Since(now))
				time.Sleep(i.Client.RetryAfter)
				goto STREAM_INVOKE_LOOP
			}

			result, err := protomap.MessageToAny(msg.ProtoReflect(), interceptors.DurationDecoder, interceptors.TimeDecoder)
			if err != nil {
				i.Log.Error("message decoding failed",
					"error", err,
				)
				i.Observe(metrics.EventFailed, time.Since(now))
				continue STREAM_READ_LOOP
			}

			event := core.NewEventWithData(string(i.method.FullName()), result)
			for k, v := range i.LabelHeaders {
				if h := header.Get(v); len(h) > 0 {
					event.SetLabel(k, strings.Join(h, "; "))
				}
			}

			i.Ider.Apply(event)
			i.Out <- event
			i.Observe(metrics.EventAccepted, time.Since(now))
		}
	}
}

func (i *DynamicGRPC) prepareServer() error {
	if i.method.IsStreamingServer() {
		return fmt.Errorf("%v is not an unary or client stream", i.Procedure)
	}

	if len(i.Server.Address) == 0 {
		return errors.New("address required")
	}

	if len(i.Server.InvokeResponse) == 0 {
		return errors.New("invoke response required")
	}

	i.respMsg = dynamicpb.NewMessage(i.method.Output())
	if err := protojson.Unmarshal([]byte(i.Server.InvokeResponse), i.respMsg); err != nil {
		return fmt.Errorf("invoke response unmarshal failed: %w", err)
	}

	listener, err := net.Listen("tcp", i.Server.Address)
	if err != nil {
		return fmt.Errorf("error creating listener: %v", err)
	}
	i.listener = listener

	opts, err := i.Server.ServerOptions()
	if err != nil {
		return err
	}

	sd := &grpc.ServiceDesc{
		ServiceName: string(i.method.Parent().FullName()),
	}

	h := &Handler{
		BaseInput:    i.BaseInput,
		Ider:         i.Ider,
		Procedure:    i.Procedure,
		LabelHeaders: i.LabelHeaders,
		RespMsg:      i.respMsg,
		RecvMsg:      i.method.Input(),
	}

	if i.method.IsStreamingClient() {
		sd.Streams = []grpc.StreamDesc{
			{
				StreamName:    string(i.method.Name()),
				Handler:       h.HandleClientStream,
				ServerStreams: false,
				ClientStreams: true,
			},
		}
	} else {
		sd.Methods = []grpc.MethodDesc{
			{
				MethodName: string(i.method.Name()),
				Handler:    h.HandleUnary,
			},
		}
	}

	i.server = grpc.NewServer(opts...)
	i.server.RegisterService(sd, nil)

	return nil
}

func (i *DynamicGRPC) prepareClient() error {
	if i.method.IsStreamingClient() || !i.method.IsStreamingServer() {
		return fmt.Errorf("%v is not a server-side stream", i.Procedure)
	}

	if len(i.Client.Address) == 0 {
		return errors.New("address required")
	}

	if len(i.Client.InvokeRequest) == 0 {
		return errors.New("invoke request required")
	}

	i.initMsg = dynamicpb.NewMessage(i.method.Input())
	if err := protojson.Unmarshal([]byte(i.Client.InvokeRequest), i.initMsg); err != nil {
		return fmt.Errorf("invoke request unmarshal failed: %w", err)
	}

	i.initMD = make(metadata.MD)
	for k, v := range i.Client.InvokeHeaders {
		i.initMD.Set(k, v)
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
			Client: dynamicgrpc.Client{
				RetryAfter:      5 * time.Second,
				TLSClientConfig: &tls.TLSClientConfig{},
			},
			Server: dynamicgrpc.Server{
				MaxMessageSize:       4 * datasize.Mebibyte,
				NumStreamWorkers:     5,
				MaxConcurrentStreams: 5,
				TLSServerConfig:      &tls.TLSServerConfig{},
			},
		}
	})
}
