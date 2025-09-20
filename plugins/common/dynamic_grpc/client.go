package dynamicgrpc

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/gekatateam/neptunus/plugins/common/tls"
)

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

func (c Client) DialOptions() ([]grpc.DialOption, error) {
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
