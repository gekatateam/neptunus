package dynamicgrpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bufbuild/protocompile"

	"github.com/jhump/protoreflect/v2/grpcdynamic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/tls"
	"github.com/gekatateam/protomap"
	"github.com/gekatateam/protomap/interceptors"
)

const (
	modeServerSideStream = "ServerSideStream"
)

type DynamicGRPC struct {
	*core.BaseInput `mapstructure:"-"`
	Mode            string            `mapstructure:"mode"`
	ProtoFiles      []string          `mapstructure:"proto_files"`
	ImportPaths     []string          `mapstructure:"import_paths"`
	Procedure       string            `mapstructure:"procedure"`
	LabelHeaders    map[string]string `mapstructure:"labelheaders"`

	// ServerSideStream
	Client     Client `mapstructure:"client"`
	clientConn *grpc.ClientConn
	method     protoreflect.MethodDescriptor
	initMsg    proto.Message
	initMD     metadata.MD

	cancelFunc context.CancelFunc
	doneCh     chan struct{}
}

type Client struct {
	Address               string            `mapstructure:"address"`
	RetryAfter            time.Duration     `mapstructure:"retry_after"`
	InvokeRequest         string            `mapstructure:"invoke_request"`
	InvokeHeaders         map[string]string `mapstructure:"invoke_headers"`
	Authority             string            `mapstructure:"authority"`               // https://pkg.go.dev/google.golang.org/grpc#WithAuthority
	UserAgent             string            `mapstructure:"user_agent"`              // https://pkg.go.dev/google.golang.org/grpc#WithUserAgent
	InactiveTransportPing time.Duration     `mapstructure:"inactive_transport_ping"` // keepalive ClientParameters.Time
	InactiveTransportAge  time.Duration     `mapstructure:"inactive_transport_age"`  // keepalive ClientParameters.Timeout
	PermitWithoutStream   bool              `mapstructure:"permit_without_stream"`   // keepalive ClientParameters.PermitWithoutStream
	*tls.TLSClientConfig  `mapstructure:",squash"`
}

func (i *DynamicGRPC) Init() error {
	if len(i.ProtoFiles) == 0 {
		return errors.New("at least one .proto file required")
	}

	if len(i.Procedure) == 0 {
		return errors.New("procedure name required")
	}

	i.doneCh = make(chan struct{})

	switch i.Mode {
	case modeServerSideStream:
		if err := i.prepareClient(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown mode: %v", i.Mode)
	}

	return nil
}

func (i *DynamicGRPC) Close() error {
	i.cancelFunc()
	<-i.doneCh

	switch i.Mode {
	case modeServerSideStream:
		return i.clientConn.Close()
	default:
		// btw unreachable code, mode checks on init
		panic(fmt.Errorf("unknown mode: %v", i.Mode))
	}
}

func (i *DynamicGRPC) Run() {
	switch i.Mode {
	case modeServerSideStream:
		i.runServerSideStream()
	default:
		// btw unreachable code, mode checks on init
		panic(fmt.Errorf("unknown mode: %v", i.Mode))
	}

	close(i.doneCh)
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
		now := time.Now()

		select {
		case <-ctx.Done():
			i.Log.Info("server-side stream context canceled")
			return
		default:
			msg, err := ss.RecvMsg()
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
				i.Log.Error("receive from stream failed",
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

			e := core.NewEventWithData(string(i.method.FullName()), result)
			for k, v := range i.LabelHeaders {
				if h := header.Get(v); len(h) > 0 {
					e.SetLabel(k, strings.Join(h, "; "))
				}
			}

			i.Out <- e
			i.Observe(metrics.EventAccepted, time.Since(now))
		}
	}
}

func (i *DynamicGRPC) prepareClient() error {
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

	if m.IsStreamingClient() {
		return fmt.Errorf("%v is not a server-side stream", i.Procedure)
	}

	if !m.IsStreamingServer() {
		return fmt.Errorf("%v is not a server-side stream", i.Procedure)
	}

	i.method = m

	if len(i.Client.Address) == 0 {
		return errors.New("address required")
	}

	if len(i.Client.InvokeRequest) == 0 {
		return errors.New("init request required")
	}

	i.initMsg = dynamicpb.NewMessage(m.Input())
	if err := protojson.Unmarshal([]byte(i.Client.InvokeRequest), i.initMsg); err != nil {
		return fmt.Errorf("init request unmarshal failed: %w", err)
	}

	i.initMD = make(metadata.MD)
	for k, v := range i.Client.InvokeHeaders {
		i.initMD.Set(k, v)
	}

	opts, err := dialOptions(i.Client)
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

func dialOptions(c Client) ([]grpc.DialOption, error) {
	var dialOpts []grpc.DialOption

	if len(c.Authority) > 0 {
		dialOpts = append(dialOpts, grpc.WithAuthority(c.Authority))
	}

	if len(c.UserAgent) > 0 {
		dialOpts = append(dialOpts, grpc.WithUserAgent(c.UserAgent))
	}

	dialOpts = append(dialOpts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                c.InactiveTransportPing,
		Timeout:             c.InactiveTransportAge,
		PermitWithoutStream: c.PermitWithoutStream,
	}))

	tlsConfig, err := c.TLSClientConfig.Config()
	if err != nil {
		return nil, err
	}

	if tlsConfig != nil {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	}

	return dialOpts, nil
}

func init() {
	plugins.AddInput("dynamic_grpc", func() core.Input {
		return &DynamicGRPC{
			Client: Client{
				RetryAfter:      5 * time.Second,
				TLSClientConfig: &tls.TLSClientConfig{},
			},
		}
	})
}
