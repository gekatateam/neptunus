package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	common "github.com/gekatateam/neptunus/plugins/common/grpc"
	grpcstats "github.com/gekatateam/neptunus/plugins/common/metrics"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

type Grpc struct {
	alias string
	pipe  string

	EnableMetrics  bool              `mapstructure:"enable_metrics"`
	Address        string            `mapstructure:"address"`
	Procedure      string            `mapstructure:"procedure"`
	RetryAfter     time.Duration     `mapstructure:"retry_after"`
	MaxAttempts    int               `mapstructure:"max_attempts"`
	DialOptions    DialOptions       `mapstructure:"dial_options"`
	CallOptions    CallOptions       `mapstructure:"call_options"`
	MetadataLabels map[string]string `mapstructure:"metadatalabels"`

	*tls.TLSClientConfig          `mapstructure:",squash"`
	*batcher.Batcher[*core.Event] `mapstructure:",squash"`

	sendFn   func(ch <-chan *core.Event)
	client   common.InputClient
	conn     *grpc.ClientConn
	callOpts []grpc.CallOption

	in  <-chan *core.Event
	log *slog.Logger
	ser core.Serializer
}

type DialOptions struct {
	Authority             string        `mapstructure:"authority"`               // https://pkg.go.dev/google.golang.org/grpc#WithAuthority
	UserAgent             string        `mapstructure:"user_agent"`              // https://pkg.go.dev/google.golang.org/grpc#WithUserAgent
	InactiveTransportPing time.Duration `mapstructure:"inactive_transport_ping"` // keepalive ClientParameters.Time
	InactiveTransportAge  time.Duration `mapstructure:"inactive_transport_age"`  // keepalive ClientParameters.Timeout
	PermitWithoutStream   bool          `mapstructure:"permit_without_stream"`   // keepalive ClientParameters.PermitWithoutStream
}

type CallOptions struct {
	ContentSUbtype string `mapstructure:"content_subtype"` // https://pkg.go.dev/google.golang.org/grpc#WithUserAgent
	WaitForReady   bool   `mapstructure:"wait_for_ready"`  // https://pkg.go.dev/google.golang.org/grpc#WaitForReady
}

func (o *Grpc) Init(config map[string]any, alias, pipeline string, log *slog.Logger) error {
	if err := mapstructure.Decode(config, o); err != nil {
		return err
	}

	o.alias = alias
	o.pipe = pipeline
	o.log = log

	if len(o.Address) == 0 {
		return errors.New("address required")
	}

	if o.Batcher.Buffer < 0 {
		o.Batcher.Buffer = 1
	}

	tlsConfig, err := o.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	options := dialOptions(o.DialOptions)
	if o.EnableMetrics {
		options = append(options, grpc.WithStreamInterceptor(grpcstats.GrpcClientStreamInterceptor(pipeline, alias)))
		options = append(options, grpc.WithUnaryInterceptor(grpcstats.GrpcClientUnaryInterceptor(pipeline, alias)))
	}

	if tlsConfig != nil {
		options = append(options, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	}

	conn, err := grpc.Dial(o.Address, options...)
	if err != nil {
		return err
	}
	o.conn = conn
	o.client = common.NewInputClient(o.conn)
	o.callOpts = callOptions(o.CallOptions)

	switch o.Procedure {
	case "unary":
		o.sendFn = o.sendOne
	case "bulk":
		o.sendFn = o.sendBulk
	case "stream":
		o.sendFn = o.sendStream
	default:
		return fmt.Errorf("unknown procedure: %v; expected one of: unary, bulk, stream", o.Procedure)
	}
	o.log.Info(fmt.Sprintf("gRPC client works in %v mode", o.Procedure))

	return nil
}

func (o *Grpc) Prepare(in <-chan *core.Event) {
	o.in = in
}

func (o *Grpc) SetSerializer(s core.Serializer) {
	o.ser = s
}

func (o *Grpc) Run() {
	o.sendFn(o.in)
}

func (o *Grpc) Close() error {
	return o.conn.Close()
}

func (o *Grpc) Alias() string {
	return o.alias
}

func (o *Grpc) sendOne(ch <-chan *core.Event) {
MAIN_LOOP:
	for e := range ch {
		now := time.Now()
		event, err := o.ser.Serialize(e)
		if err != nil {
			o.log.Error("serialization failed",
				"error", err.Error(),
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.Done()
			metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
			continue
		}

		md := metadata.MD{}
		for k, v := range o.MetadataLabels {
			if label, ok := e.GetLabel(v); ok {
				md.Append(k, label)
			}
		}

		var attempts int = 1
		for {
			_, err = o.client.SendOne(
				metadata.NewOutgoingContext(context.Background(), md),
				&common.Data{Data: event},
				o.callOpts...,
			)

			if err == nil {
				o.log.Debug("event sent",
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				e.Done()
				metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventAccepted, time.Since(now))
				continue MAIN_LOOP
			}

			switch {
			case o.MaxAttempts > 0 && attempts < o.MaxAttempts:
				o.log.Warn(fmt.Sprintf("unary call attempt %v of %v failed", attempts, o.MaxAttempts))
				attempts++
				time.Sleep(o.RetryAfter)
			case o.MaxAttempts > 0 && attempts >= o.MaxAttempts:
				o.log.Error(fmt.Sprintf("unary call failed after %v attemtps", attempts),
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				e.Done()
				metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
				continue MAIN_LOOP
			default:
				o.log.Error("unary call failed",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				time.Sleep(o.RetryAfter)
			}
		}
	}
}

func (o *Grpc) sendBulk(ch <-chan *core.Event) {
	o.Batcher.Run(ch, func(buf []*core.Event) {
		if len(buf) == 0 {
			return
		}

		var stream common.Input_SendBulkClient

	MAIN_LOOP:
		for _, e := range buf {
			now := time.Now()
			event, err := o.ser.Serialize(e)
			if err != nil {
				o.log.Error("serialization failed",
					"error", err.Error(),
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				e.Done()
				metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
				continue
			}

			for {
				if stream == nil {
					stream, err = o.newBulkStream()
					if err != nil {
						o.log.Error("event sending failed",
							"error", err,
							slog.Group("event",
								"id", e.Id,
								"key", e.RoutingKey,
							),
						)
						e.Done()
						metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
						continue MAIN_LOOP
					}
				}

				err = stream.Send(&common.Data{
					Data: event,
				})

				if err == nil {
					o.log.Debug("event sent",
						slog.Group("event",
							"id", e.Id,
							"key", e.RoutingKey,
						),
					)
					e.Done()
					metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventAccepted, time.Since(now))
					continue MAIN_LOOP
				}

				stream = nil // if error occured, stream is already aborted, so we need to reopen it
				o.log.Error("sending to bulk stream failed",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				time.Sleep(o.RetryAfter)
			}
		}

		if stream == nil { // stream may be dead after failed attempts
			o.log.Warn("summary receiving failed beacuse the stream is already dead")
			return
		}

		sum, err := stream.CloseAndRecv()
		if err != nil {
			o.log.Warn("all events has been sent, but summary receiving failed",
				"error", err,
			)
			return
		}

		o.log.Debug(fmt.Sprintf("accepted: %v, failed: %v", sum.Accepted, sum.Failed))
	})
}

func (o *Grpc) sendStream(ch <-chan *core.Event) {
	doneCh := make(chan struct{})
	var stream common.Input_SendStreamClient

MAIN_LOOP:
	for {
		select {
		case <-doneCh:
			o.log.Info("cancel signal received, closing stream")
			stream.CloseSend()
			stream = nil
			time.Sleep(o.RetryAfter)
		case e, ok := <-ch:
			if !ok { // no events left
				if stream != nil { // stream may be dead after failed attempts
					stream.CloseSend()
				}
				return
			}

			now := time.Now()
			event, err := o.ser.Serialize(e)
			if err != nil {
				o.log.Error("serialization failed",
					"error", err.Error(),
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				e.Done()
				metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
				continue MAIN_LOOP
			}

			for {
				if stream == nil {
					stream, err = o.newInternalStream(doneCh)
					if err != nil {
						o.log.Error("event sending failed",
							"error", err,
							slog.Group("event",
								"id", e.Id,
								"key", e.RoutingKey,
							),
						)
						e.Done()
						metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
						continue MAIN_LOOP
					}
				}

				err := stream.Send(&common.Event{
					RoutingKey: e.RoutingKey,
					Labels:     e.Labels,
					Data:       event,
					Tags:       e.Tags,
					Id:         e.Id,
					Errors:     e.Errors.Slice(),
					Timestamp:  e.Timestamp.Format(time.RFC3339Nano),
				})

				if err == nil {
					o.log.Debug("event sent",
						slog.Group("event",
							"id", e.Id,
							"key", e.RoutingKey,
						),
					)
					e.Done()
					metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventAccepted, time.Since(now))
					continue MAIN_LOOP
				}

				stream = nil // if error occured, stream is already aborted, so we need to reopen it
				o.log.Error("sending to internal stream failed",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				time.Sleep(o.RetryAfter)
			}
		}
	}
}

func (o *Grpc) newInternalStream(cancelCh chan<- struct{}) (common.Input_SendStreamClient, error) {
	var stream common.Input_SendStreamClient
	var err error
	var attempts int = 1
	for {
		stream, err = o.client.SendStream(context.Background(), o.callOpts...)
		if err == nil {
			o.log.Info("internal stream opened")

			go func() {
				// handle cancel signal from server and close the stream
				// it's not for gracefull shutdown, but for gracefull disconnect
				// from stopping server without data loss
				// after this, client will try to reopen the stream in main loop
				_, err := stream.Recv()
				if err == nil {
					cancelCh <- struct{}{}
				}

				o.log.Debug("receiving goroutine returns")
			}()

			return stream, nil
		}

		switch {
		case o.MaxAttempts > 0 && attempts < o.MaxAttempts:
			o.log.Warn(fmt.Sprintf("internal stream open attempt %v of %v failed", attempts, o.MaxAttempts))
			attempts++
			time.Sleep(o.RetryAfter)
		case o.MaxAttempts > 0 && attempts >= o.MaxAttempts:
			o.log.Error(fmt.Sprintf("internal stream open failed after %v attemtps", attempts),
				"error", err,
			)
			return nil, err
		default:
			o.log.Error("internal stream open failed",
				"error", err,
			)
			time.Sleep(o.RetryAfter)
		}
	}
}

func (o *Grpc) newBulkStream() (common.Input_SendBulkClient, error) {
	var stream common.Input_SendBulkClient
	var err error
	var attempts int = 1
	for {
		stream, err = o.client.SendBulk(context.Background(), o.callOpts...)
		if err == nil {
			o.log.Debug("bulk stream opened")
			return stream, nil
		}

		switch {
		case o.MaxAttempts > 0 && attempts < o.MaxAttempts:
			o.log.Warn(fmt.Sprintf("bulk stream open attempt %v of %v failed", attempts, o.MaxAttempts))
			attempts++
			time.Sleep(o.RetryAfter)
		case o.MaxAttempts > 0 && attempts >= o.MaxAttempts:
			o.log.Error(fmt.Sprintf("bulk stream open failed after %v attemtps", attempts),
				"error", err,
			)
			return nil, err
		default:
			o.log.Error("bulk stream open failed",
				"error", err,
			)
			time.Sleep(o.RetryAfter)
		}
	}
}

func dialOptions(opts DialOptions) []grpc.DialOption {
	var dialOpts []grpc.DialOption

	if len(opts.Authority) > 0 {
		dialOpts = append(dialOpts, grpc.WithAuthority(opts.Authority))
	}

	if len(opts.UserAgent) > 0 {
		dialOpts = append(dialOpts, grpc.WithUserAgent(opts.UserAgent))
	}

	dialOpts = append(dialOpts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                opts.InactiveTransportPing,
		Timeout:             opts.InactiveTransportAge,
		PermitWithoutStream: opts.PermitWithoutStream,
	}))

	dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	return dialOpts
}

func callOptions(opts CallOptions) []grpc.CallOption {
	var callOpts []grpc.CallOption

	if len(opts.ContentSUbtype) > 0 {
		callOpts = append(callOpts, grpc.CallContentSubtype(opts.ContentSUbtype))
	}

	callOpts = append(callOpts, grpc.WaitForReady(opts.WaitForReady))

	return callOpts
}

func init() {
	plugins.AddOutput("grpc", func() core.Output {
		return &Grpc{
			RetryAfter: 5 * time.Second,
			Batcher: &batcher.Batcher[*core.Event]{
				Buffer:   100,
				Interval: 5 * time.Second,
			},
			TLSClientConfig: &tls.TLSClientConfig{},
		}
	})
}
