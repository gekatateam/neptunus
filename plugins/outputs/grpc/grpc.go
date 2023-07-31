package grpc

import (
	"context"
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

	Address string        `mapstructure:"address"`
	Method  string        `mapstructure:"method"`
	Timeout time.Duration `mapstructure:"timeout"`

	sendFn func(ch <-chan *core.Event)
	client common.InputClient
	conn   *grpc.ClientConn

	in  <-chan *core.Event
	log logger.Logger
	ser core.Serializer
}

func (o *Grpc) Init(config map[string]any, alias, pipeline string, log logger.Logger) error {
	if err := mapstructure.Decode(config, o); err != nil {
		return err
	}

	o.alias = alias
	o.pipe = pipeline
	o.log = log

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
			_, err = o.client.SendOne(context.Background(), &common.Data{Data: event})
			if err == nil {
				break
			}
			o.log.Errorf("unary call failed: %v", err.Error())
			metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
			time.Sleep(o.Timeout)
		}

		metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventAccepted, time.Since(now))
	}
}

func (o *Grpc) sendBulk(ch <-chan *core.Event) {}

func (o *Grpc) openStream(ch <-chan *core.Event) {
	stream := o.newStream()

	for e := range ch {
		now := time.Now()
		event, err := o.ser.Serialize(e)
		if err != nil {
			o.log.Errorf("serialization failed: %v", err.Error())
			metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventFailed, time.Since(now))
			continue
		}

		for {
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

			o.log.Errorf("sending to stream failed: %v; event id: %v", err.Error(), e.Id)
			time.Sleep(o.Timeout)

			if err == io.EOF { // reconnect
				stream = o.newStream()
			}
		}
		metrics.ObserveOutputSummary("grpc", o.alias, o.pipe, metrics.EventAccepted, time.Since(now))
	}

	stream.CloseAndRecv()
}

func (o *Grpc) newStream() common.Input_OpenStreamClient {
	var stream common.Input_OpenStreamClient
	var err error
	for {
		stream, err = o.client.OpenStream(context.Background())
		if err == nil {
			o.log.Debug("stream opened")
			break
		}

		o.log.Errorf("stream open failed: %v", err.Error())
		time.Sleep(o.Timeout)
	}
	return stream
}

func init() {
	plugins.AddOutput("grpc", func() core.Output {
		return &Grpc{
			Timeout: 5 * time.Second,
		}
	})
}
