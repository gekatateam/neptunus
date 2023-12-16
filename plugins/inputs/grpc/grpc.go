package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
	common "github.com/gekatateam/neptunus/plugins/common/grpc"
	"github.com/gekatateam/neptunus/plugins/common/ider"
	grpcstats "github.com/gekatateam/neptunus/plugins/common/metrics"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

type Grpc struct {
	alias                string
	pipe                 string
	EnableMetrics        bool              `mapstructure:"enable_metrics"`
	Address              string            `mapstructure:"address"`
	ServerOptions        ServerOptions     `mapstructure:"server_options"`
	LabelMetadata        map[string]string `mapstructure:"labelmetadata"`
	*ider.Ider           `mapstructure:",squash"`
	*tls.TLSServerConfig `mapstructure:",squash"`

	server   *grpc.Server
	listener net.Listener
	closeCh  chan struct{}

	log    *slog.Logger
	out    chan<- *core.Event
	parser core.Parser

	sendOneDesc  string
	sendBulkDesc string

	common.UnimplementedInputServer
}

type ServerOptions struct {
	// main server options
	// TODO: add r/w buffers ortions
	MaxMessageSize       int    `mapstructure:"max_message_size"`
	NumStreamWorkers     uint32 `mapstructure:"num_stream_workers"`
	MaxConcurrentStreams uint32 `mapstructure:"max_concurrent_streams"`

	// keepalive server options
	MaxConnectionIdle     time.Duration `mapstructure:"max_connection_idle"`     // ServerParameters.MaxConnectionIdle
	MaxConnectionAge      time.Duration `mapstructure:"max_connection_age"`      // ServerParameters.MaxConnectionAge
	MaxConnectionGrace    time.Duration `mapstructure:"max_connection_grace"`    // ServerParameters.MaxConnectionAgeGrace
	InactiveTransportPing time.Duration `mapstructure:"inactive_transport_ping"` // ServerParameters.Time
	InactiveTransportAge  time.Duration `mapstructure:"inactive_transport_age"`  // ServerParameters.Timeout
}

func (i *Grpc) Alias() string {
	return i.alias
}

func (i *Grpc) Close() error {
	i.log.Debug("closing plugin")
	close(i.closeCh)
	i.server.GracefulStop()
	return nil
}

func (i *Grpc) Init(config map[string]any, alias string, pipeline string, log *slog.Logger) error {
	if err := mapstructure.Decode(config, i); err != nil {
		return err
	}

	i.alias = alias
	i.pipe = pipeline
	i.log = log
	i.closeCh = make(chan struct{})

	if err := i.Ider.Init(); err != nil {
		return err
	}

	if len(i.Address) == 0 {
		return errors.New("address required")
	}

	tlsConfig, err := i.TLSServerConfig.Config()
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", i.Address)
	if err != nil {
		return fmt.Errorf("error creating listener: %v", err)
	}
	i.listener = listener

	options := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(i.ServerOptions.MaxMessageSize),
		grpc.MaxSendMsgSize(i.ServerOptions.MaxMessageSize),
		grpc.NumStreamWorkers(i.ServerOptions.NumStreamWorkers),
		grpc.MaxConcurrentStreams(i.ServerOptions.MaxConcurrentStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     i.ServerOptions.MaxConnectionIdle,
			MaxConnectionAge:      i.ServerOptions.MaxConnectionAge,
			MaxConnectionAgeGrace: i.ServerOptions.MaxConnectionGrace,
			Time:                  i.ServerOptions.InactiveTransportPing,
			Timeout:               i.ServerOptions.InactiveTransportAge,
		}),
	}

	if i.EnableMetrics {
		options = append(options, grpc.StreamInterceptor(grpcstats.GrpcServerStreamInterceptor(pipeline, alias)))
		options = append(options, grpc.UnaryInterceptor(grpcstats.GrpcServerUnaryInterceptor(pipeline, alias)))
	}

	if tlsConfig != nil {
		options = append(options, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}

	i.server = grpc.NewServer(options...)
	common.RegisterInputServer(i.server, i)

	i.sendOneDesc = "/" + common.Input_ServiceDesc.ServiceName + "/" + common.Input_ServiceDesc.Methods[0].MethodName
	i.sendBulkDesc = "/" + common.Input_ServiceDesc.ServiceName + "/" + common.Input_ServiceDesc.Streams[0].StreamName

	return nil
}

func (i *Grpc) Prepare(out chan<- *core.Event) {
	i.out = out
}

func (i *Grpc) Run() {
	i.log.Info(fmt.Sprintf("starting grpc server on %v", i.Address))
	if err := i.server.Serve(i.listener); err != nil {
		i.log.Error("grpc server startup failed",
			"error", err.Error(),
		)
	} else {
		i.log.Info("grpc server stopped")
	}
}

func (i *Grpc) SetParser(p core.Parser) {
	i.parser = p
}

func (i *Grpc) SendOne(ctx context.Context, data *common.Data) (*common.Nil, error) {
	now := time.Now()
	p, _ := peer.FromContext(ctx)
	i.log.Debug("request received",
		"sender", p.Addr.String(),
	)

	events, err := i.unpackData(ctx, data, i.sendOneDesc)
	if err != nil {
		i.log.Error("data unpack failed",
			"error", err,
		)
		metrics.ObserveInputSummary("grpc", i.alias, i.pipe, metrics.EventFailed, time.Since(now))
		return &common.Nil{}, status.Error(codes.InvalidArgument, err.Error())
	}

	for _, e := range events {
		i.out <- e
		i.log.Debug("event accepted",
			slog.Group("event",
				"id", e.Id,
				"key", e.RoutingKey,
			),
		)
		metrics.ObserveInputSummary("grpc", i.alias, i.pipe, metrics.EventAccepted, time.Since(now))
		now = time.Now()
	}

	return &common.Nil{}, nil
}

func (i *Grpc) SendBulk(stream common.Input_SendBulkServer) error {
	p, _ := peer.FromContext(stream.Context())
	i.log.Debug("bulk stream accepted",
		"sender", p.Addr.String(),
	)

	sum := &common.BulkSummary{
		Accepted: 0,
		Failed:   0,
		Errors:   make(map[int32]string),
	}

	for {
		data, err := stream.Recv()
		if err == io.EOF {
			stream.SendAndClose(sum)
			break
		}
		now := time.Now()

		if err != nil {
			i.log.Error("receiving from stream failed",
				"error", err,
			)
			metrics.ObserveInputSummary("grpc", i.alias, i.pipe, metrics.EventFailed, time.Since(now))
			return err
		}

		events, err := i.unpackData(stream.Context(), data, i.sendBulkDesc)
		if err != nil {
			sum.Errors[sum.Failed+sum.Accepted] = err.Error()
			sum.Failed++
			i.log.Warn("data unpack failed",
				"error", err,
			)
			continue
		}

		sum.Accepted++
		for _, e := range events {
			i.out <- e
			i.log.Debug("event accepted",
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			metrics.ObserveInputSummary("grpc", i.alias, i.pipe, metrics.EventAccepted, time.Since(now))
			now = time.Now()
		}
	}

	i.log.Debug("stream ended")
	return nil
}

func (i *Grpc) SendStream(stream common.Input_SendStreamServer) error {
	p, _ := peer.FromContext(stream.Context())
	i.log.Debug("internal stream accepted",
		"sender", p.Addr.String(),
	)

	doneCh := make(chan struct{})
	stopCh := make(chan struct{})

	go func() {
		select {
		case <-i.closeCh:
			// send close signal to client
			// there is no need to handle an error because
			// the stream may be already aborted
			i.log.Debug("plugin closing, sending cancellation token")
			stream.Send(&common.Cancel{})
		case <-stopCh:
			i.log.Debug("internal stream done")
		}
		close(doneCh)
		i.log.Debug("internal stream sending goroutine returns")
	}()

	for {
		event, err := stream.Recv()
		if err == io.EOF {
			close(stopCh)
			break
		}
		now := time.Now()

		if err != nil {
			i.log.Error("receiving from stream failed",
				"error", err,
			)
			metrics.ObserveInputSummary("grpc", i.alias, i.pipe, metrics.EventFailed, time.Since(now))
			close(stopCh)
			return err
		}

		e, err := i.unpackEvent(event)
		if err != nil {
			i.log.Warn("event unpack failed",
				"error", err,
			)
			metrics.ObserveInputSummary("grpc", i.alias, i.pipe, metrics.EventFailed, time.Since(now))
			continue
		}

		i.out <- e
		i.log.Debug("event accepted",
			slog.Group("event",
				"id", e.Id,
				"key", e.RoutingKey,
			),
		)
		metrics.ObserveInputSummary("grpc", i.alias, i.pipe, metrics.EventAccepted, time.Since(now))
	}

	<-doneCh
	i.log.Debug("internal stream closed")
	return nil
}

func (i *Grpc) unpackData(ctx context.Context, data *common.Data, defaultkey string) ([]*core.Event, error) {
	events, err := i.parser.Parse(data.GetData(), defaultkey)
	if err != nil {
		return nil, fmt.Errorf("data parsing failed: %w", err)
	}

	for _, e := range events {
		p, _ := peer.FromContext(ctx)
		e.AddLabel("input", "grpc")
		e.AddLabel("server", i.Address)
		e.AddLabel("sender", p.Addr.String())

		md, _ := metadata.FromIncomingContext(ctx)
		for k, v := range i.LabelMetadata {
			if val, ok := md[v]; ok {
				e.AddLabel(k, strings.Join(val, ";"))
			}
		}

		i.Ider.Apply(e)
	}

	return events, nil
}

func (i *Grpc) unpackEvent(event *common.Event) (*core.Event, error) {
	events, err := i.parser.Parse(event.GetData(), event.GetRoutingKey())
	if err != nil {
		return nil, fmt.Errorf("data parsing failed: %w", err)
	}
	e := events[0]

	timestamp, err := time.Parse(time.RFC3339Nano, event.GetTimestamp())
	if err != nil {
		return nil, fmt.Errorf("timestamp parsing failed: %w", err)
	}
	e.Timestamp = timestamp

	e.Id = event.GetId()

	for k, v := range event.GetLabels() {
		e.AddLabel(k, v)
	}

	for _, v := range event.GetTags() {
		e.AddTag(v)
	}

	for _, v := range event.GetErrors() {
		e.StackError(errors.New(v))
	}

	return e, nil
}

func init() {
	plugins.AddInput("grpc", func() core.Input {
		return &Grpc{
			Address: ":5800",
			ServerOptions: ServerOptions{
				MaxMessageSize:        4 * 1024 * 1024, // 4 MiB
				NumStreamWorkers:      5,
				MaxConcurrentStreams:  5,
				MaxConnectionIdle:     0,
				MaxConnectionAge:      0,
				MaxConnectionGrace:    0,
				InactiveTransportPing: 0,
				InactiveTransportAge:  0,
			},
			Ider:            &ider.Ider{},
			TLSServerConfig: &tls.TLSServerConfig{},
		}
	})
}
