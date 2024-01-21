package elasticsearch

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/operationtype"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	"github.com/gekatateam/neptunus/plugins/common/pool"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

type Elasticsearch struct {
	alias                  string
	pipe                   string
	Servers                []string      `mapstructure:"servers"`
	Username               string        `mapstructure:"username"`
	Password               string        `mapstructure:"password"`
	ServiceToken           string        `mapstructure:"service_token"` // Service token for authorization; if set, overrides username/password.
	APIKey                 string        `mapstructure:"api_key"`       // Base64-encoded token for authorization; if set, overrides username/password and service token.
	CloudID                string        `mapstructure:"cloud_id"`
	CertificateFingerprint string        `mapstructure:"cert_fingerprint"` // SHA256 hex fingerprint given by Elasticsearch on first launch.
	EnableCompression      bool          `mapstructure:"enable_compression"`
	DiscoverInterval       time.Duration `mapstructure:"-"`
	RequestTimeout         time.Duration `mapstructure:"request_timeout"`
	IdleTimeout            time.Duration `mapstructure:"idle_timeout"`
	PipelineLabel          string        `mapstructure:"pipeline_label"`
	RoutingLabel           string        `mapstructure:"routing_label"`
	DataOnly               bool          `mapstructure:"data_only"`
	Operation              string        `mapstructure:"operation"`
	MaxAttempts            int           `mapstructure:"max_attempts"`
	RetryAfter             time.Duration `mapstructure:"retry_after"`

	*tls.TLSClientConfig          `mapstructure:",squash"`
	*batcher.Batcher[*core.Event] `mapstructure:",squash"`

	client       *elasticsearch.TypedClient
	indexersPool *pool.Pool[*core.Event]

	in  <-chan *core.Event
	log *slog.Logger
}

func (o *Elasticsearch) Init(config map[string]any, alias, pipeline string, log *slog.Logger) error {
	if err := mapstructure.Decode(config, o); err != nil {
		return err
	}

	o.alias = alias
	o.pipe = pipeline
	o.log = log

	if len(o.Servers) == 0 {
		return errors.New("at least one server url required")
	}

	switch o.Operation {
	case "create", "index":
	default:
		return fmt.Errorf("unknown operation: %v; expected one of: create, index", o.Operation)
	}

	tlsConfig, err := o.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	client, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Addresses:               o.Servers,
		Username:                o.Username,
		Password:                o.Password,
		ServiceToken:            o.ServiceToken,
		APIKey:                  o.APIKey,
		CloudID:                 o.CloudID,
		CertificateFingerprint:  o.CertificateFingerprint,
		CompressRequestBody:     o.EnableCompression,
		DiscoverNodesInterval:   o.DiscoverInterval,
		DiscoverNodesOnStart:    false,
		EnableDebugLogger:       false, // <-
		DisableRetry:            true,
		EnableCompatibilityMode: true,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Logger: &TransportLogger{
			log: log,
		},
	})
	if err != nil {
		return err
	}

	o.client = client
	o.indexersPool = pool.New(o.newIndexer)

	return nil
}

func (o *Elasticsearch) SetChannels(in <-chan *core.Event) {
	o.in = in
}

func (o *Elasticsearch) Run() {
	clearTicker := time.NewTicker(time.Minute)
	if o.IdleTimeout == 0 {
		clearTicker.Stop()
	}

MAIN_LOOP:
	for {
		select {
		case e, ok := <-o.in:
			if !ok {
				clearTicker.Stop()
				break MAIN_LOOP
			}
			o.indexersPool.Get(o.pipeline(e)).Push(e)
		case <-clearTicker.C:
			for _, pipeline := range o.indexersPool.Keys() {
				if time.Since(o.indexersPool.Get(pipeline).LastWrite()) > o.IdleTimeout {
					o.indexersPool.Remove(pipeline)
				}
			}
		}
	}
}

func (o *Elasticsearch) Close() error {
	o.client = nil
	return nil
}

func (o *Elasticsearch) newIndexer(pipeline string) pool.Runner[*core.Event] {
	return &indexer{
		alias:        o.alias,
		pipe:         o.pipe,
		lastWrite:    time.Now(),
		pipeline:     pipeline,
		dataOnly:     o.DataOnly,
		operation:    operationtype.OperationType{Name: o.Operation},
		routingLabel: o.RoutingLabel,
		timeout:      o.RequestTimeout,
		maxAttempts:  o.MaxAttempts,
		retryAfter:   o.RetryAfter,
		client:       o.client,
		log:          o.log,
		Batcher:      o.Batcher,
		input:        make(chan *core.Event),
	}
}

func (o *Elasticsearch) pipeline(e *core.Event) string {
	if len(o.PipelineLabel) > 0 {
		pipe, _ := e.GetLabel(o.PipelineLabel)
		return pipe
	}
	return ""
}

func init() {
	plugins.AddOutput("elasticsearch", func() core.Output {
		return &Elasticsearch{
			EnableCompression: true,
			RequestTimeout:    10 * time.Second,
			IdleTimeout:       1 * time.Hour,
			DataOnly:          true,
			Operation:         "create",
			RetryAfter:        5 * time.Second,
			Batcher: &batcher.Batcher[*core.Event]{
				Buffer:   100,
				Interval: 5 * time.Second,
			},
			TLSClientConfig: &tls.TLSClientConfig{},
		}
	})
}
