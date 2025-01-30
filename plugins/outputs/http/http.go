package http

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	"github.com/gekatateam/neptunus/plugins/common/pool"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

type Http struct {
	*core.BaseOutput `mapstructure:"-"`
	Host             string        `mapstructure:"host"`
	Method           string        `mapstructure:"method"`
	Timeout          time.Duration `mapstructure:"timeout"`
	IdleConnTimeout  time.Duration `mapstructure:"idle_conn_timeout"`
	MaxIdleConns     int           `mapstructure:"max_idle_conns"`
	IdleTimeout      time.Duration `mapstructure:"idle_timeout"`
	RetryableCodes   []int         `mapstructure:"retryable_codes"`

	Headerlabels map[string]string `mapstructure:"headerlabels"`
	Paramfields  map[string]string `mapstructure:"paramfields"`

	*tls.TLSClientConfig          `mapstructure:",squash"`
	*batcher.Batcher[*core.Event] `mapstructure:",squash"`
	*retryer.Retryer              `mapstructure:",squash"`

	requestersPool *pool.Pool[*core.Event]
	retryableCodes map[int]struct{}
	providedUri    *url.URL

	client *http.Client
	ser    core.Serializer
}

func (o *Http) Init() error {
	if len(o.Host) == 0 {
		return errors.New("host required")
	}

	url, err := url.ParseRequestURI(o.Host)
	if err != nil {
		return err
	}

	o.providedUri = url

	if o.Batcher.Buffer < 0 {
		o.Batcher.Buffer = 1
	}

	retryableCodes := map[int]struct{}{}
	for _, v := range o.RetryableCodes {
		retryableCodes[v] = struct{}{}
	}
	o.retryableCodes = retryableCodes

	tlsConfig, err := o.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	o.client = &http.Client{
		Timeout: o.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
			IdleConnTimeout: o.IdleConnTimeout,
			MaxIdleConns:    o.MaxIdleConns,
		},
	}

	o.requestersPool = pool.New(o.newRequester)

	return nil
}

func (o *Http) SetSerializer(s core.Serializer) {
	o.ser = s
}

func (o *Http) Run() {
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
			o.requestersPool.Get(o.uriFromRoutingKey(e.RoutingKey)).Push(e)
		case <-clearTicker.C:
			for _, pipeline := range o.requestersPool.Keys() {
				if time.Since(o.requestersPool.Get(pipeline).LastWrite()) > o.IdleTimeout {
					o.requestersPool.Remove(pipeline)
				}
			}
		}
	}
}

func (o *Http) Close() error {
	o.requestersPool.Close()
	o.client.CloseIdleConnections()
	o.client = nil
	return nil
}

func (o *Http) newRequester(uri string) pool.Runner[*core.Event] {
	return &requester{
		BaseOutput:     o.BaseOutput,
		lastWrite:      time.Now(),
		uri:            uri,
		method:         o.Method,
		retryableCodes: o.retryableCodes,
		headerlabels:   o.Headerlabels,
		paramfields:    o.Paramfields,
		client:         o.client,
		ser:            o.ser,
		Batcher:        o.Batcher,
		Retryer:        o.Retryer,
		input:          make(chan *core.Event),
	}
}

func (o *Http) uriFromRoutingKey(rk string) string {
	return o.providedUri.JoinPath(rk).String()
}

func init() {
	plugins.AddOutput("http", func() core.Output {
		return &Http{
			IdleTimeout: 1 * time.Hour,
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
