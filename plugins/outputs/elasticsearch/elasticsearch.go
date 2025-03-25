package elasticsearch

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	"github.com/gekatateam/neptunus/plugins/common/pool"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

type Elasticsearch struct {
	*core.BaseOutput       `mapstructure:"-"`
	URLs                   []string      `mapstructure:"urls"`
	Username               string        `mapstructure:"username"`
	Password               string        `mapstructure:"password"`
	ServiceToken           string        `mapstructure:"service_token"` // Service token for authorization; if set, overrides username/password.
	APIKey                 string        `mapstructure:"api_key"`       // Base64-encoded token for authorization; if set, overrides username/password and service token.
	CloudID                string        `mapstructure:"cloud_id"`
	CertificateFingerprint string        `mapstructure:"cert_fingerprint"` // SHA256 hex fingerprint given by Elasticsearch on first launch.
	EnableCompression      bool          `mapstructure:"enable_compression"`
	DiscoverInterval       time.Duration `mapstructure:"discover_interval"`
	RequestTimeout         time.Duration `mapstructure:"request_timeout"`
	IdleTimeout            time.Duration `mapstructure:"idle_timeout"`
	PipelineLabel          string        `mapstructure:"pipeline_label"`
	RoutingLabel           string        `mapstructure:"routing_label"`
	DataOnly               bool          `mapstructure:"data_only"`
	Operation              string        `mapstructure:"operation"`

	*tls.TLSClientConfig          `mapstructure:",squash"`
	*batcher.Batcher[*core.Event] `mapstructure:",squash"`
	*retryer.Retryer              `mapstructure:",squash"`

	client       *elasticsearch.Client
	indexersPool *pool.Pool[*core.Event]
}

func (o *Elasticsearch) Init() error {
	if len(o.URLs) == 0 {
		return errors.New("at least one server url required")
	}

	switch o.Operation {
	case "create", "index":
	default:
		return fmt.Errorf("unknown operation: %v; expected one of: create, index", o.Operation)
	}

	if o.IdleTimeout > 0 && o.IdleTimeout < time.Minute {
		o.IdleTimeout = time.Minute
	}

	if o.Batcher.Buffer < 0 {
		o.Batcher.Buffer = 1
	}

	tlsConfig, err := o.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses:               o.URLs,
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
			log: o.Log,
		},
	})
	if err != nil {
		return err
	}

	o.client = client
	o.indexersPool = pool.New(o.newIndexer)

	return nil
}

func (o *Elasticsearch) Run() {
	clearTicker := time.NewTicker(time.Minute)
	if o.IdleTimeout == 0 {
		clearTicker.Stop()
	}

MAIN_LOOP:
	for {
		select {
		case e, ok := <-o.In:
			if !ok {
				clearTicker.Stop()
				break MAIN_LOOP
			}
			o.indexersPool.Get(o.pipeline(e)).Push(e)
		case <-clearTicker.C:
			for _, pipeline := range o.indexersPool.Keys() {
				if time.Since(o.indexersPool.LastWrite(pipeline)) > o.IdleTimeout {
					o.indexersPool.Remove(pipeline)
				}
			}
		}
	}
}

func (o *Elasticsearch) Close() error {
	o.indexersPool.Close()
	o.client = nil
	return nil
}

func (o *Elasticsearch) newIndexer(pipeline string) pool.Runner[*core.Event] {
	return &indexer{
		BaseOutput:   o.BaseOutput,
		pipeline:     pipeline,
		dataOnly:     o.DataOnly,
		operation:    o.Operation,
		routingLabel: o.RoutingLabel,
		timeout:      o.RequestTimeout,
		client:       o.client,
		Batcher:      o.Batcher,
		Retryer:      o.Retryer,
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
			Batcher: &batcher.Batcher[*core.Event]{
				Buffer:   100,
				Interval: 5 * time.Second,
			},
			TLSClientConfig: &tls.TLSClientConfig{},
			Retryer: &retryer.Retryer{
				RetryAttempts: 0,
				RetryAfter:    5 * time.Second,
			},
		}
	})
}
