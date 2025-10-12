package http

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
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
	Fallbacks        []string      `mapstructure:"fallbacks"`
	Timeout          time.Duration `mapstructure:"timeout"`
	IdleConnTimeout  time.Duration `mapstructure:"idle_conn_timeout"`
	MaxIdleConns     int           `mapstructure:"max_idle_conns"`
	IdleTimeout      time.Duration `mapstructure:"idle_timeout"`
	SuccessCodes     []int         `mapstructure:"success_codes"`
	SuccessBody      string        `mapstructure:"success_body"`
	MethodLabel      string        `mapstructure:"method_label"`

	Headerlabels map[string]string `mapstructure:"headerlabels"`
	Paramfields  map[string]string `mapstructure:"paramfields"`

	*tls.TLSClientConfig          `mapstructure:",squash"`
	*batcher.Batcher[*core.Event] `mapstructure:",squash"`
	*retryer.Retryer              `mapstructure:",squash"`

	requestersPool *pool.Pool[*core.Event, string]
	successCodes   map[int]struct{}
	successBody    *regexp.Regexp
	providedUri    *url.URL
	fallbacks      []*url.URL

	client *http.Client
	ser    core.Serializer
}

func (o *Http) Init() error {
	if len(o.Host) == 0 {
		return errors.New("host required")
	}

	if len(o.Method) == 0 {
		return errors.New("method required")
	}

	if len(o.SuccessCodes) == 0 {
		return errors.New("at least one success code required")
	}

	uri, err := url.ParseRequestURI(o.Host)
	if err != nil {
		return err
	}

	o.providedUri = uri

	o.fallbacks = make([]*url.URL, 0, len(o.Fallbacks))
	for _, f := range o.Fallbacks {
		uri, err := url.ParseRequestURI(f)
		if err != nil {
			return fmt.Errorf("fallback %v: %w", f, err)
		}

		o.fallbacks = append(o.fallbacks, uri)
	}

	if o.Batcher.Buffer <= 0 {
		o.Batcher.Buffer = 1
	}

	if o.IdleTimeout > 0 && o.IdleTimeout < time.Minute {
		o.IdleTimeout = time.Minute
	}

	successCodes := map[int]struct{}{}
	for _, v := range o.SuccessCodes {
		successCodes[v] = struct{}{}
	}
	o.successCodes = successCodes

	o.successBody = nil
	if len(o.SuccessBody) > 0 {
		rex, err := regexp.Compile(o.SuccessBody)
		if err != nil {
			return fmt.Errorf("successBody regexp: %w", err)
		}
		o.successBody = rex
	}

	tlsConfig, err := o.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	o.client = &http.Client{
		Timeout: o.Timeout,
		Transport: &http.Transport{
			TLSClientConfig:   tlsConfig,
			IdleConnTimeout:   o.IdleConnTimeout,
			MaxIdleConns:      o.MaxIdleConns,
			ForceAttemptHTTP2: tlsConfig != nil,
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
			o.requestersPool.Get(e.RoutingKey).Push(e)
		case <-clearTicker.C:
			for _, pipeline := range o.requestersPool.Keys() {
				if time.Since(o.requestersPool.LastWrite(pipeline)) > o.IdleTimeout {
					o.requestersPool.Remove(pipeline)
				}
			}
		}
	}

	o.requestersPool.Close()
}

func (o *Http) Close() error {
	o.ser.Close()
	o.client.CloseIdleConnections()
	return nil
}

func (o *Http) newRequester(path string) pool.Runner[*core.Event] {
	return &requester{
		BaseOutput:   o.BaseOutput,
		method:       o.Method,
		methodLabel:  o.MethodLabel,
		successBody:  o.successBody,
		successCodes: o.successCodes,
		headerlabels: o.Headerlabels,
		paramfields:  o.Paramfields,
		client:       o.client,
		ser:          o.ser,
		Batcher:      o.Batcher,
		Retryer:      o.Retryer,
		input:        make(chan *core.Event),
		uri:          o.uriFromRoutingKey(path),
		fallbacks:    o.fallbacksFromRoutingKey(path),
	}
}

func (o *Http) uriFromRoutingKey(rk string) *url.URL {
	return o.providedUri.JoinPath(rk)
}

func (o *Http) fallbacksFromRoutingKey(rk string) []*url.URL {
	u := make([]*url.URL, 0, len(o.fallbacks))
	for _, f := range o.fallbacks {
		u = append(u, f.JoinPath(rk))
	}
	return u
}

func init() {
	plugins.AddOutput("http", func() core.Output {
		return &Http{
			Method:          http.MethodPost,
			SuccessCodes:    []int{200, 201, 204},
			Timeout:         10 * time.Second,
			IdleConnTimeout: 1 * time.Minute,
			MaxIdleConns:    10,
			IdleTimeout:     1 * time.Hour,
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
