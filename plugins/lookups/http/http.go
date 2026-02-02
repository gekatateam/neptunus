package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
	"github.com/gekatateam/neptunus/plugins/common/tls"
	"github.com/gekatateam/neptunus/plugins/core/lookup"
)

type Http struct {
	*core.BaseLookup `mapstructure:"-"`
	Host             string            `mapstructure:"host"`
	Fallbacks        []string          `mapstructure:"fallbacks"`
	Method           string            `mapstructure:"method"`
	Timeout          time.Duration     `mapstructure:"timeout"`
	IdleConnTimeout  time.Duration     `mapstructure:"idle_conn_timeout"`
	MaxIdleConns     int               `mapstructure:"max_idle_conns"`
	SuccessCodes     []int             `mapstructure:"success_codes"`
	SuccessBody      string            `mapstructure:"success_body"`
	RequestBody      string            `mapstructure:"request_body"`
	RequestQuery     string            `mapstructure:"request_query"`
	Headers          map[string]string `mapstructure:"headers"`

	*tls.TLSClientConfig `mapstructure:",squash"`
	*retryer.Retryer     `mapstructure:",squash"`

	body         []byte
	headers      http.Header
	successCodes map[int]struct{}
	successBody  *regexp.Regexp
	baseUrl      *url.URL
	fallbacks    []*url.URL

	client *http.Client
	parser core.Parser
}

func (l *Http) Init() error {
	if len(l.Host) == 0 {
		return errors.New("host required")
	}

	if len(l.Method) == 0 {
		return errors.New("method required")
	}

	if len(l.SuccessCodes) == 0 {
		return errors.New("at least one success code required")
	}

	l.body = []byte(l.RequestBody)

	l.headers = make(http.Header, len(l.Headers))
	for k, v := range l.Headers {
		l.headers.Set(k, v)
	}

	uri, err := url.ParseRequestURI(l.Host)
	if err != nil {
		return err
	}
	l.baseUrl = uri
	l.baseUrl.RawQuery = l.RequestQuery

	l.fallbacks = make([]*url.URL, 0, len(l.Fallbacks))
	for _, f := range l.Fallbacks {
		uri, err := url.ParseRequestURI(f)
		if err != nil {
			return fmt.Errorf("fallback %v: %w", f, err)
		}

		uri.RawQuery = l.RequestQuery
		l.fallbacks = append(l.fallbacks, uri)
	}

	successCodes := map[int]struct{}{}
	for _, v := range l.SuccessCodes {
		successCodes[v] = struct{}{}
	}
	l.successCodes = successCodes

	l.successBody = nil
	if len(l.SuccessBody) > 0 {
		rex, err := regexp.Compile(l.SuccessBody)
		if err != nil {
			return fmt.Errorf("successBody regexp: %w", err)
		}
		l.successBody = rex
	}

	tlsConfig, err := l.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	l.client = &http.Client{
		Timeout: l.Timeout,
		Transport: &http.Transport{
			TLSClientConfig:   tlsConfig,
			IdleConnTimeout:   l.IdleConnTimeout,
			MaxIdleConns:      l.MaxIdleConns,
			ForceAttemptHTTP2: tlsConfig != nil,
		},
	}

	return nil
}

func (l *Http) Close() error {
	l.client.CloseIdleConnections()
	return l.parser.Close()
}

func (l *Http) SetParser(p core.Parser) {
	l.parser = p
}

func (l *Http) Update() (any, error) {
	content, err := l.performWithFallback()
	if err != nil {
		return nil, err
	}

	event, err := l.parser.Parse(content, "")
	if err != nil {
		return nil, err
	}

	if len(event) == 0 {
		return nil, errors.New("parser returns zero events, nothing to store")
	}

	if len(event) > 1 {
		l.Log.Warn("parser returns more than one event, only first event will be used for lookup data")
	}

	return event[0].Data, nil
}

func (l *Http) performWithFallback() ([]byte, error) {
	rawResponse, err := l.perform(l.baseUrl, l.Method, l.body, l.headers)

	if err != nil && len(l.fallbacks) > 0 {
		l.Log.Warn("request failed, trying to perform to fallback",
			"error", err,
		)

		for _, f := range l.fallbacks {
			rawResponse, err = l.perform(f, l.Method, l.body, l.headers)
			if err == nil {
				l.Log.Warn(fmt.Sprintf("request to %v succeeded, but it is still a fallback", f.String()))
				return rawResponse, nil
			}

			l.Log.Warn(fmt.Sprintf("request to fallback %v failed", f.String()),
				"error", err,
			)
		}
	}

	return rawResponse, err
}

func (l *Http) perform(uri *url.URL, method string, body []byte, header http.Header) ([]byte, error) {
	var rawResponse []byte

	err := l.Retryer.Do("perform request", l.Log, func() error {
		req, err := http.NewRequest(method, uri.String(), bytes.NewReader(body))
		if err != nil {
			return err
		}

		req.Header = header

		l.Log.Debug(fmt.Sprintf("request body: %v; request headers: %v; request query: %v", string(body), header, uri.RawQuery))

		res, err := l.client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		rawBody, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("request performed, but body reading failed: %w", err)
		}
		rawResponse = rawBody

		if _, ok := l.successCodes[res.StatusCode]; ok {
			l.Log.Debug(fmt.Sprintf("request performed successfully with code: %v, body: %v", res.StatusCode, string(rawBody)))
			return nil
		} else {
			if l.successBody != nil && l.successBody.Match(rawBody) {
				l.Log.Debug(fmt.Sprintf("request performed with code: %v, body: %v; "+
					"but response body matches configured regexp", res.StatusCode, string(rawBody)))
				return nil
			}

			return fmt.Errorf("request result not successful with code: %v, body: %v", res.StatusCode, string(rawBody))
		}
	})

	return rawResponse, err
}

func init() {
	plugins.AddLookup("http", func() core.Lookup {
		return &lookup.Lookup{
			LazyLookup: &Http{
				Method:          http.MethodPost,
				SuccessCodes:    []int{200, 201, 204},
				Timeout:         10 * time.Second,
				IdleConnTimeout: 1 * time.Minute,
				MaxIdleConns:    10,
				TLSClientConfig: &tls.TLSClientConfig{},
				Retryer: &retryer.Retryer{
					RetryAttempts: 0,
					RetryAfter:    5 * time.Second,
				},
			},
			Interval: 30 * time.Second,
		}
	})
}
