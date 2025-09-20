package dynamicgrpc

import (
	"time"

	"kythe.io/kythe/go/util/datasize"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/gekatateam/neptunus/plugins/common/tls"
)

type Server struct {
	Address               string        `mapstructure:"address"`
	InvokeResponse        string        `mapstructure:"invoke_response"`
	MaxMessageSize        datasize.Size `mapstructure:"max_message_size"`
	NumStreamWorkers      uint32        `mapstructure:"num_stream_workers"`
	MaxConcurrentStreams  uint32        `mapstructure:"max_concurrent_streams"`
	MaxConnectionIdle     time.Duration `mapstructure:"max_connection_idle"`     // ServerParameters.MaxConnectionIdle
	MaxConnectionAge      time.Duration `mapstructure:"max_connection_age"`      // ServerParameters.MaxConnectionAge
	MaxConnectionGrace    time.Duration `mapstructure:"max_connection_grace"`    // ServerParameters.MaxConnectionAgeGrace
	InactiveTransportPing time.Duration `mapstructure:"inactive_transport_ping"` // ServerParameters.Time
	InactiveTransportAge  time.Duration `mapstructure:"inactive_transport_age"`  // ServerParameters.Timeout
	*tls.TLSServerConfig  `mapstructure:",squash"`
}

func (s Server) ServerOptions() ([]grpc.ServerOption, error) {
	var serverOptions []grpc.ServerOption

	serverOptions = append(serverOptions,
		grpc.MaxRecvMsgSize(int(s.MaxMessageSize.Bytes())),
		grpc.MaxSendMsgSize(int(s.MaxMessageSize.Bytes())),
		grpc.NumStreamWorkers(s.NumStreamWorkers),
		grpc.MaxConcurrentStreams(s.MaxConcurrentStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     s.MaxConnectionIdle,
			MaxConnectionAge:      s.MaxConnectionAge,
			MaxConnectionAgeGrace: s.MaxConnectionGrace,
			Time:                  s.InactiveTransportPing,
			Timeout:               s.InactiveTransportAge,
		}),
	)

	tlsConfig, err := s.TLSServerConfig.Config()
	if err != nil {
		return nil, err
	}

	if tlsConfig != nil {
		serverOptions = append(serverOptions, grpc.Creds(credentials.NewTLS(tlsConfig)))
	} else {
		serverOptions = append(serverOptions, grpc.Creds(insecure.NewCredentials()))

	}

	return serverOptions, nil
}
