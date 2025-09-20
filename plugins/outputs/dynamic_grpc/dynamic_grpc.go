package dynamicgrpc

// as CLIENT:
// - if unary - send every event
// - if client stream - send all events in batch
//
// as SERVER
// - if server stream - send every event to all connected clients

import (
	"context"
	"fmt"
	"time"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/grpc"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	dynamicgrpc "github.com/gekatateam/neptunus/plugins/common/dynamic_grpc"
	"github.com/gekatateam/neptunus/plugins/common/pool"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

const (
	modeAsClient = "AsClient"
)

type DynamicGRPC struct {
	*core.BaseOutput `mapstructure:"-"`
	Mode             string            `mapstructure:"mode"`
	ProtoFiles       []string          `mapstructure:"proto_files"`
	ImportPaths      []string          `mapstructure:"import_paths"`
	LabelHeaders     map[string]string `mapstructure:"labelheaders"`

	// AsClient
	Client     dynamicgrpc.Client `mapstructure:"client"`
	clientConn *grpc.ClientConn

	*batcher.Batcher[*core.Event] `mapstructure:",squash"`
	*retryer.Retryer              `mapstructure:",squash"`

	callersPool *pool.Pool[*core.Event, string]
	resolver    linker.Resolver
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

func (o *DynamicGRPC) Init() error {
	compiler := &protocompile.Compiler{
		Resolver: protocompile.CompositeResolver{
			protocompile.WithStandardImports(&protocompile.SourceResolver{}),
			&protocompile.SourceResolver{ImportPaths: o.ImportPaths},
		},
	}

	f, err := compiler.Compile(context.Background(), o.ProtoFiles...)
	if err != nil {
		return fmt.Errorf("compilation error: %w", err)
	}

	o.resolver = f.AsResolver()

	switch o.Mode {
	case modeAsClient:

	default:
		return fmt.Errorf("unknown mode: %v", o.Mode)
	}

	return nil
}

func (o *DynamicGRPC) Run() {
	for e := range o.In {
		now := time.Now()

		o.Done <- e
		o.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func (o *DynamicGRPC) Close() error {
	return nil
}

func (o *DynamicGRPC) newCaller(rpc string) pool.Runner[*core.Event] {
	return &Caller{}
}

func init() {
	plugins.AddOutput("dynamic_grpc", func() core.Output {
		return &DynamicGRPC{
			Client: dynamicgrpc.Client{
				RetryAfter:      5 * time.Second,
				TLSClientConfig: &tls.TLSClientConfig{},
			},
		}
	})
}
