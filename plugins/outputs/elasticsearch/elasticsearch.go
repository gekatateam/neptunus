package elasticsearch

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	"github.com/gekatateam/neptunus/plugins/common/tls"
	// common "github.com/gekatateam/neptunus/plugins/common/elasticsearch"
)

type Elasticsearch struct {
	alias string
	pipe  string
	Servers []string `mapstructure:"servers"`
	Username  string `mapstructure:"username"`
    Password  string `mapstructure:"password"`
	ServiceToken           string `mapstructure:"service_token"` // Service token for authorization; if set, overrides username/password.
    APIKey                 string `mapstructure:"api_key"`// Base64-encoded token for authorization; if set, overrides username/password and service token.
    CloudID                string `mapstructure:"cloud_id"`
    CertificateFingerprint string `mapstructure:"cert_fingerprint"` // SHA256 hex fingerprint given by Elasticsearch on first launch.
	EnableCompression bool `mapstructure:"enable_compression"`
	DiscoverInterval time.Duration `mapstructure:"discover_interval"`
	PipelineLabel string `mapstructure:"pipeline_label"`
	DataOnly bool `mapstructure:"data_only"`

	*tls.TLSClientConfig          `mapstructure:",squash"`
	*batcher.Batcher[*core.Event] `mapstructure:",squash"`

	client *elasticsearch.Client

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

	tlsConfig, err := o.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: o.Servers,
		Username: o.Username,
		Password: o.Password,
		ServiceToken: o.ServiceToken,
		APIKey: o.APIKey,
		CloudID: o.CloudID,
		CertificateFingerprint: o.CertificateFingerprint,
		CompressRequestBody: o.EnableCompression,
		DiscoverNodesInterval: o.DiscoverInterval,
		DisableRetry: true,
		EnableCompatibilityMode: true,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	})
	if err != nil {
		return err
	}

	o.client = client

	return nil
}

func (o *Elasticsearch) SetChannels(in <-chan *core.Event) {
	o.in = in
}

func (o *Elasticsearch) Run() {
	for e := range o.in {
		now := time.Now()
		o.log.Error("serialization failed",
			slog.Group("event",
				"id", e.Id,
				"key", e.RoutingKey,
			),
		)

		e.Done()
		metrics.ObserveOutputSummary("elasticsearch", o.alias, o.pipe, metrics.EventAccepted, time.Since(now))
	}
}

func (o *Elasticsearch) Close() error {
	o.client = nil
	return nil
}

func init() {
	plugins.AddOutput("elasticsearch", func() core.Output {
		return &Elasticsearch{

			Batcher: &batcher.Batcher[*core.Event]{
				Buffer:   100,
				Interval: 5 * time.Second,
			},
			TLSClientConfig: &tls.TLSClientConfig{},
		}
	})
}
