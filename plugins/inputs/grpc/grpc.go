package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
	common "github.com/gekatateam/neptunus/plugins/common/grpc"
)

var _ common.InputServer = &Grpc{}

type Grpc struct {
	alias   string
	pipe    string
	Address string `mapstructure:"address"`

	// main server options
	// TODO: add r/w buffers ortions
	MaxMessageSize       int `mapstructure:"max_message_size"`
	NumStreamWorkers     int `mapstructure:"num_stream_workers"`
	MaxConcurrentStreams int `mapstructure:"max_concurrent_streams"`

	// keepalive server options
	MaxConnectionIdle     time.Duration `mapstructure:"max_connection_idle"`      // ServerParameters.MaxConnectionIdle
	MaxConnectionAge      time.Duration `mapstructure:"max_connection_age"`       // ServerParameters.MaxConnectionAge
	MaxConnectionAgeGrace time.Duration `mapstructure:"max_connection_age_grace"` // ServerParameters.MaxConnectionAgeGrace
	InactiveTransportPing time.Duration `mapstructure:"inactive_transport_idle"`  // ServerParameters.Time
	InactiveTransportAge  time.Duration `mapstructure:"inactive_transport_age"`   // ServerParameters.Timeout

	server   *grpc.Server
	listener net.Listener

	log    logger.Logger
	out    chan<- *core.Event
	parser core.Parser

	common.UnimplementedInputServer
}

func (i *Grpc) Alias() string {
	return i.alias
}

func (i *Grpc) Close() error {
	i.server.GracefulStop()
	return nil
}

func (i *Grpc) Init(config map[string]any, alias string, pipeline string, log logger.Logger) error {
	if err := mapstructure.Decode(config, i); err != nil {
		return err
	}

	i.alias = alias
	i.pipe = pipeline
	i.log = log

	if len(i.Address) == 0 {
		return errors.New("address required")
	}

	listener, err := net.Listen("tcp", i.Address)
	if err != nil {
		return fmt.Errorf("error creating listener: %v", err)
	}
	i.listener = listener

	return nil
}

func (i *Grpc) Prepare(out chan<- *core.Event) {
	i.out = out
}

func (i *Grpc) Run() {
	i.log.Infof("starting grpc server on %v", i.Address)
	if err := i.server.Serve(i.listener); err != nil {
		i.log.Errorf("grpc server startup failed: %v", err.Error())
	} else {
		i.log.Debug("grpc server stopped")
	}
}

func (i *Grpc) SetParser(p core.Parser) {
	i.parser = p
}

func (i *Grpc) SendBulk(common.Input_SendBulkServer) error
func (i *Grpc) SendOne(context.Context, *common.Event) (*common.Nil, error)
func (i *Grpc) OpenStream(common.Input_OpenStreamServer) error

func init() {
	plugins.AddInput("httpl", func() core.Input {
		return &Grpc{
			Address: ":9800",
		}
	})
}
