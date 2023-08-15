package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	common "github.com/gekatateam/neptunus/plugins/common/grpc"
)

type Grpc struct {
	alias string
	pipe  string

	Address        string            `mapstructure:"address"`
	Procedure      string            `mapstructure:"procedure"`
	Sleep          time.Duration     `mapstructure:"sleep"`
	Buffer         int               `mapstructure:"buffer"`
	Interval       time.Duration     `mapstructure:"interval"`
	MaxAttempts    int               `mapstructure:"max_attempts"`
	DialOptions    DialOptions       `mapstructure:"dial_options"`
	CallOptions    CallOptions       `mapstructure:"call_options"`
	MetadataLabels map[string]string `mapstructure:"metadatalabels"`

	sendFn   func(ch <-chan *core.Event)
	client   common.InputClient
	conn     *grpc.ClientConn
	callOpts []grpc.CallOption

	in  <-chan *core.Event
	log logger.Logger
	ser core.Serializer
	b   *batcher.Batcher
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

func (o *Grpc) Init(config map[string]any, alias, pipeline string, log logger.Logger) error {
	if err := mapstructure.Decode(config, o); err != nil {
		return err
	}

	o.alias = alias
	o.pipe = pipeline
	o.log = log

	if len(o.Address) == 0 {
		return errors.New("address required")
	}

	var err error
	o.conn, err = grpc.Dial(o.Address, dialOptions(o.DialOptions)...)
	if err != nil {
		return err
	}
	o.client = common.NewInputClient(o.conn)
	o.callOpts = callOptions(o.CallOptions)

	switch o.Procedure {
	case "one":
		o.sendFn = o.sendOne
	case "bulk":
		o.sendFn = o.sendBulk
	case "stream":
		o.sendFn = o.sendStream
	default:
		return fmt.Errorf("unknown procedure: %v; expected one of: one, bulk, stream", o.Procedure)
	}

	o.b = &batcher.Batcher{
		Buffer:   o.Buffer,
		Interval: o.Interval,
	}

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
	for e := range ch {
		now := time.Now()
		event, err := o.ser.Serialize(e)
		if err != nil {
			o.log.Errorf("serialization failed: %v", err.Error())
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
				o.log.Debugf("sent event id: %v", e.Id)
				metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventAccepted, time.Since(now))
				break
			}

			switch {
			case o.MaxAttempts > 0 && attempts < o.MaxAttempts:
				o.log.Debugf("unary call attempt %v of %v failed", attempts, o.MaxAttempts)
				attempts++
				time.Sleep(o.Sleep)
			case o.MaxAttempts > 0 && attempts >= o.MaxAttempts:
				o.log.Errorf("unary call failed after %v attemtps: %v", attempts, err.Error())
				metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
				goto SEND_LOOP_OUT
			default:
				o.log.Errorf("unary call failed: %v", err.Error())
				time.Sleep(o.Sleep)
			}
		}
	SEND_LOOP_OUT:
	}
}

func (o *Grpc) sendBulk(ch <-chan *core.Event) {
	o.b.Run(ch, func(buf []*core.Event) {
		stream := o.newBulkStream()

		for _, e := range buf {
			now := time.Now()
			event, err := o.ser.Serialize(e)
			if err != nil {
				o.log.Errorf("serialization failed: %v", err.Error())
				metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
				continue
			}

			for {
				if stream == nil {
					stream = o.newBulkStream()
				}

				err = stream.Send(&common.Data{
					Data: event,
				})

				if err == nil {
					o.log.Debugf("sent event id: %v", e.Id)
					break
				}

				// io.EOF means than stream is dead on server side
				if err == io.EOF {
					stream = nil
				}

				o.log.Errorf("sending to stream failed: %v; event id: %v", err.Error(), e.Id)
				time.Sleep(o.Sleep)
			}

			metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventAccepted, time.Since(now))
		}

		sum, err := stream.CloseAndRecv()
		if err != nil {
			o.log.Errorf("all events have been sent, but summary receiving failed: %v", err.Error())
			return
		}
		o.log.Debugf("accepted: %v, failed: %v", sum.Accepted, sum.Failed)
	})
}

func (o *Grpc) sendStream(ch <-chan *core.Event) {
	stream := o.newInternalStream()
	doneCh := make(chan struct{})
	stopCh := make(chan struct{}, 1)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			select {
			case <-stopCh:
				o.log.Debug("receiving goroutine returns")
				return
			default:
			}

			// handle cancel signal from server and close the stream
			// it's not for gracefull shutdown, but for gracefull disconnect
			// from stopping server without data loss
			// after this, client will try to reopen the stream in main loop
			if stream != nil {
				_, err := stream.Recv()
				if err == nil {
					doneCh <- struct{}{}
				}
			}
		}
	}()

	for e := range ch {
		select {
		case <-doneCh:
			o.log.Info("cancel signal received, closing stream")
			stream.CloseSend()
			stream = nil
			time.Sleep(o.Sleep)
		default:
		}

		now := time.Now()
		event, err := o.ser.Serialize(e)
		if err != nil {
			o.log.Errorf("serialization failed: %v", err.Error())
			metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
			continue
		}

		for {
			if stream == nil {
				stream = o.newInternalStream()
			}

			err := stream.Send(&common.Event{
				RoutingKey: e.RoutingKey,
				Labels:     e.Labels,
				Data:       event,
				Tags:       e.Tags,
				Id:         e.Id.String(),
				Errors:     e.Errors.Slice(),
				Timestamp:  e.Timestamp.Format(time.RFC3339Nano),
			})
			if err == nil {
				o.log.Debugf("sent event id: %v", e.Id)
				break
			}

			// io.EOF means than stream is dead on server side
			if err == io.EOF {
				stream = nil
			}

			o.log.Errorf("sending to stream failed: %v; event id: %v", err.Error(), e.Id)
			time.Sleep(o.Sleep)
		}
		metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventAccepted, time.Since(now))
	}

	stopCh <- struct{}{}
	stream.CloseSend()
	wg.Wait()
}

func (o *Grpc) newInternalStream() common.Input_SendStreamClient {
	var stream common.Input_SendStreamClient
	var err error
	for {
		stream, err = o.client.SendStream(context.Background(), o.callOpts...)
		if err == nil {
			o.log.Info("internal stream opened")
			break
		}

		o.log.Errorf("internal stream open failed: %v", err.Error())
		time.Sleep(o.Sleep)
	}
	return stream
}

func (o *Grpc) newBulkStream() common.Input_SendBulkClient {
	var stream common.Input_SendBulkClient
	var err error
	for {
		stream, err = o.client.SendBulk(context.Background(), o.callOpts...)
		if err == nil {
			o.log.Debug("bulk stream opened")
			break
		}

		o.log.Errorf("bulk stream open failed: %v", err.Error())
		time.Sleep(o.Sleep)
	}
	return stream
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
			Sleep:    5 * time.Second,
			Buffer:   100,
			Interval: 5 * time.Second,
		}
	})
}
