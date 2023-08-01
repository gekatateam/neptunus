package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
	common "github.com/gekatateam/neptunus/plugins/common/grpc"
)

type Grpc struct {
	alias string
	pipe  string

	Address     string        `mapstructure:"address"`
	Method      string        `mapstructure:"method"`
	Sleep       time.Duration `mapstructure:"sleep"`
	DialOptions DialOptions   `mapstructure:"dial_options"`
	CallOptions CallOptions   `mapstructure:"call_options"`

	sendFn func(ch <-chan *core.Event)
	client common.InputClient
	conn   *grpc.ClientConn

	in  <-chan *core.Event
	log logger.Logger
	ser core.Serializer
}

type DialOptions struct{}

type CallOptions struct{}

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
	o.conn, err = grpc.Dial(o.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	o.client = common.NewInputClient(o.conn)

	switch o.Method {
	case "one":
		o.sendFn = o.sendOne
	case "bulk":
		o.sendFn = o.sendBulk
	case "stream":
		o.sendFn = o.openStream
	default:
		return fmt.Errorf("unknown method: %v; expected one of: one, bulk, stream", o.Method)
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

		for {
			_, err = o.client.SendOne(context.Background(), &common.Data{Data: event}, grpc.WaitForReady(true))
			if err == nil {
				o.log.Debugf("sent event id: %v", e.Id)
				break
			}
			o.log.Errorf("unary call failed: %v", err.Error())
			metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
			time.Sleep(o.Sleep)
		}

		metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventAccepted, time.Since(now))
	}
}

func (o *Grpc) sendBulk(ch <-chan *core.Event) {
	buf := make([]*core.Event, 0, 100)
	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case e, ok := <-ch:
			if !ok { // channel closed
				ticker.Stop()
				// release the buffer
				return
			}

			buf = append(buf, e)
			if len(buf) == cap(buf) {
				// do stuff
			}

			ticker.Reset(10 * time.Second)
		case <-ticker.C:

		}
	}
}

func (o *Grpc) openStream(ch <-chan *core.Event) {
	var stream common.Input_OpenStreamClient

	for e := range ch {
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

	stream.CloseAndRecv()
}

func (o *Grpc) newInternalStream() common.Input_OpenStreamClient {
	var stream common.Input_OpenStreamClient
	var err error
	for {
		stream, err = o.client.OpenStream(context.Background())
		if err == nil {
			o.log.Debug("internal stream opened")
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
		stream, err = o.client.SendBulk(context.Background())
		if err == nil {
			o.log.Debug("bulk stream opened")
			break
		}

		o.log.Errorf("bulk stream open failed: %v", err.Error())
		time.Sleep(o.Sleep)
	}
	return stream
}

func init() {
	plugins.AddOutput("grpc", func() core.Output {
		return &Grpc{
			Sleep: 5 * time.Second,
		}
	})
}
