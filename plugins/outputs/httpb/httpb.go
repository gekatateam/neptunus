package httpb

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	"github.com/gekatateam/neptunus/plugins/common/pool"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

type Httpb struct {
	*core.BaseOutput  `mapstructure:"-"`
	Host              string        `mapstructure:"host"`
	Method            string        `mapstructure:"method"`
	Timeout           time.Duration `mapstructure:"timeout"`
	IdleConnTimeout   time.Duration `mapstructure:"idle_conn_timeout"`
	MaxIdleConns      int           `mapstructure:"max_idle_conns"`
	NonRetryableCodes []int         `mapstructure:"non_retryable_codes"`

	Headerlabels map[string]string `mapstructure:"headerlabels"`

	*tls.TLSClientConfig          `mapstructure:",squash"`
	*batcher.Batcher[*core.Event] `mapstructure:",squash"`
	*retryer.Retryer              `mapstructure:",squash"`

	requestersPool *pool.Pool[*core.Event]

	client *http.Client
	ser    core.Serializer
}

func (o *Httpb) Init() error {
	if len(o.Host) == 0 {
		return errors.New("host required")
	}

	if o.Batcher.Buffer < 0 {
		o.Batcher.Buffer = 1
	}

	tlsConfig, err := o.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	o.client = &http.Client{
		Timeout: o.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
			IdleConnTimeout: o.IdleConnTimeout,
			MaxIdleConns: o.MaxIdleConns,
		},
	}

	return nil
}

func (o *Httpb) SetSerializer(s core.Serializer) {
	o.ser = s
}

func (o *Httpb) Run() {
	for e := range o.In {
		now := time.Now()
		_, err := o.ser.Serialize(e)
		if err != nil {
			o.Log.Error("serialization failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.Done()
			o.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		e.Done()
		o.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func (o *Httpb) Close() error {
	o.requestersPool.Close()
	o.client = nil
	return nil
}

func init() {
	plugins.AddOutput("httpb", func() core.Output {
		return &Httpb{
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
