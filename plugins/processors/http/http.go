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
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/convert"
	"github.com/gekatateam/neptunus/plugins/common/elog"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
	"github.com/gekatateam/neptunus/plugins/common/sharedstorage"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

var clientStorage = sharedstorage.New[*http.Client]()

type Http struct {
	*core.BaseProcessor `mapstructure:"-"`
	Host                string        `mapstructure:"host"`
	Fallbacks           []string      `mapstructure:"fallbacks"`
	Method              string        `mapstructure:"method"`
	Timeout             time.Duration `mapstructure:"timeout"`
	IdleConnTimeout     time.Duration `mapstructure:"idle_conn_timeout"`
	MaxIdleConns        int           `mapstructure:"max_idle_conns"`
	SuccessCodes        []int         `mapstructure:"success_codes"`
	SuccessBody         string        `mapstructure:"success_body"`
	PathLabel           string        `mapstructure:"path_label"`
	MethodLabel         string        `mapstructure:"method_label"`

	Headerlabels map[string]string `mapstructure:"headerlabels"`
	Labelheaders map[string]string `mapstructure:"labelheaders"`
	Paramfields  map[string]string `mapstructure:"paramfields"`

	RequestBodyFrom string `mapstructure:"request_body_from"`
	ResponseBodyTo  string `mapstructure:"response_body_to"`
	ResponseCodeTo  string `mapstructure:"response_code_to"`

	*tls.TLSClientConfig `mapstructure:",squash"`
	*retryer.Retryer     `mapstructure:",squash"`

	successCodes map[int]struct{}
	successBody  *regexp.Regexp
	baseUrl      *url.URL
	fallbacks    []*url.URL
	id           uint64

	client *http.Client
	ser    core.Serializer
	parser core.Parser
}

func (p *Http) Init() error {
	if len(p.Host) == 0 {
		return errors.New("host required")
	}

	if len(p.Method) == 0 {
		return errors.New("method required")
	}

	if len(p.SuccessCodes) == 0 {
		return errors.New("at least one success code required")
	}

	uri, err := url.ParseRequestURI(p.Host)
	if err != nil {
		return err
	}
	p.baseUrl = uri

	p.fallbacks = make([]*url.URL, 0, len(p.Fallbacks))
	for _, f := range p.Fallbacks {
		uri, err := url.ParseRequestURI(f)
		if err != nil {
			return fmt.Errorf("fallback %v: %w", f, err)
		}

		p.fallbacks = append(p.fallbacks, uri)
	}

	successCodes := map[int]struct{}{}
	for _, v := range p.SuccessCodes {
		successCodes[v] = struct{}{}
	}
	p.successCodes = successCodes

	p.successBody = nil
	if len(p.SuccessBody) > 0 {
		rex, err := regexp.Compile(p.SuccessBody)
		if err != nil {
			return fmt.Errorf("successBody regexp: %w", err)
		}
		p.successBody = rex
	}

	tlsConfig, err := p.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	p.client = clientStorage.CompareAndStore(p.id, &http.Client{
		Timeout: p.Timeout,
		Transport: &http.Transport{
			TLSClientConfig:   tlsConfig,
			IdleConnTimeout:   p.IdleConnTimeout,
			MaxIdleConns:      p.MaxIdleConns,
			ForceAttemptHTTP2: tlsConfig != nil,
		},
	})

	return nil
}

func (p *Http) Close() error {
	p.ser.Close()
	p.parser.Close()
	if clientStorage.Leave(p.id) {
		p.client.CloseIdleConnections()
	}
	return nil
}

func (p *Http) SetSerializer(s core.Serializer) {
	p.ser = s
}

func (p *Http) SetParser(parser core.Parser) {
	p.parser = parser
}

func (p *Http) SetId(id uint64) {
	p.id = id
}

func (p *Http) Run() {
	for e := range p.In {
		now := time.Now()

		header := make(http.Header)
		for k, v := range p.Headerlabels {
			if label, ok := e.GetLabel(v); ok {
				header.Add(k, label)
			}
		}

		values, err := p.unpackQueryValues(e)
		if err != nil {
			p.Log.Error("query params prepation failed",
				"error", err,
				elog.EventGroup(e),
			)
			e.StackError(err)
			p.Out <- e
			p.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		var reqBody []byte
		if len(p.RequestBodyFrom) > 0 {
			field, err := e.GetField(p.RequestBodyFrom)
			if err != nil {
				p.Log.Error("request body preparation failed",
					"error", fmt.Errorf("no such field: %v", p.RequestBodyFrom),
					elog.EventGroup(e),
				)
				e.StackError(fmt.Errorf("no such field: %v", p.RequestBodyFrom))
				p.Out <- e
				p.Observe(metrics.EventFailed, time.Since(now))
				continue
			}

			// create origin event clone, then replace it's data with one extracted before
			event := e.Clone()
			event.Data = field

			reqBody, err = p.ser.Serialize(event)
			if err != nil {
				p.Log.Error("request body preparation failed",
					"error", err,
					elog.EventGroup(e),
				)
				e.StackError(err)
				p.Out <- e
				p.Drop <- event // drop cloned event, it's useless after serialization
				p.Observe(metrics.EventFailed, time.Since(now))
				continue
			}

			p.Drop <- event // drop cloned event, it's useless after serialization
		}

		var path string
		if len(p.PathLabel) > 0 {
			label, ok := e.GetLabel(p.PathLabel)
			if !ok {
				p.Log.Error("request path preparation failed",
					"error", fmt.Errorf("event does not contains %v label", p.PathLabel),
					elog.EventGroup(e),
				)
				e.StackError(fmt.Errorf("event does not contains %v label", p.PathLabel))
				p.Out <- e
				p.Observe(metrics.EventFailed, time.Since(now))
				continue
			}
			path = label
		}

		respBody, respHeaders, statusCode, err := p.performWithFallback(path, p.requestMethod(e), values, reqBody, header)
		if err != nil {
			p.Log.Error("request failed",
				"error", err,
				elog.EventGroup(e),
			)
			e.StackError(err)
			p.Out <- e
			p.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		if len(p.ResponseCodeTo) > 0 {
			if err := e.SetField(p.ResponseCodeTo, statusCode); err != nil {
				p.Log.Error("response processing failed",
					"error", fmt.Errorf("error set response code: %w", err),
					elog.EventGroup(e),
				)
				e.StackError(fmt.Errorf("error set response code: %w", err))
				p.Out <- e
				p.Observe(metrics.EventFailed, time.Since(now))
				continue
			}
		}

		if len(respBody) > 0 && len(p.ResponseBodyTo) > 0 {
			events, err := p.parser.Parse(respBody, "")
			if err != nil {
				p.Log.Error("response processing failed",
					"error", err,
					elog.EventGroup(e),
				)
				e.StackError(err)
				p.Out <- e
				p.Observe(metrics.EventFailed, time.Since(now))
				continue
			}

			var result []any
			for _, event := range events {
				result = append(result, event.Data)
				p.Drop <- event
			}

			if err := e.SetField(p.ResponseBodyTo, result); err != nil {
				p.Log.Error("response processing failed",
					"error", fmt.Errorf("error set response data: %w", err),
					elog.EventGroup(e),
				)
				e.StackError(fmt.Errorf("error set response data: %w", err))
				p.Out <- e
				p.Observe(metrics.EventFailed, time.Since(now))
				continue
			}
		}

		for k, v := range p.Labelheaders {
			h := respHeaders.Get(v)
			if len(h) > 0 {
				e.SetLabel(k, h)
			}
		}

		p.Log.Debug("event processed",
			elog.EventGroup(e),
		)
		p.Out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func (p *Http) performWithFallback(path, method string, params url.Values, body []byte, header http.Header) ([]byte, http.Header, int, error) {
	rawResponse, headers, statusCode, err := p.perform(p.baseUrl.JoinPath(path), method, params, body, header)

	if err != nil && len(p.fallbacks) > 0 {
		p.Log.Warn("request failed, trying to perform to fallback",
			"error", err,
		)

		for _, f := range p.fallbacks {
			rawResponse, headers, statusCode, err = p.perform(f.JoinPath(path), method, params, body, header)
			if err == nil {
				p.Log.Warn(fmt.Sprintf("request to %v succeeded, but it is still a fallback", f.String()))
				return rawResponse, headers, statusCode, nil
			}

			p.Log.Warn(fmt.Sprintf("request to fallback %v failed", f.String()),
				"error", err,
			)
		}
	}

	return rawResponse, headers, statusCode, err
}

func (p *Http) perform(uri *url.URL, method string, params url.Values, body []byte, header http.Header) ([]byte, http.Header, int, error) {
	uri.RawQuery = params.Encode()

	var (
		rawResponse []byte
		headers     http.Header
		statusCode  int
	)

	err := p.Retryer.Do("perform request", p.Log, func() error {
		req, err := http.NewRequest(method, uri.String(), bytes.NewReader(body))
		if err != nil {
			return err
		}

		req.Header = header

		p.Log.Debug(fmt.Sprintf("request body: %v; request headers: %v; request query: %v", string(body), header, uri.RawQuery))

		res, err := p.client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		rawBody, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("request performed, but body reading failed: %w", err)
		}
		rawResponse = rawBody
		headers = res.Header
		statusCode = res.StatusCode

		if _, ok := p.successCodes[res.StatusCode]; ok {
			p.Log.Debug(fmt.Sprintf("request performed successfully with code: %v, body: %v", res.StatusCode, string(rawBody)))
			return nil
		} else {
			if p.successBody != nil && p.successBody.Match(rawBody) {
				p.Log.Debug(fmt.Sprintf("request performed with code: %v, body: %v; "+
					"but response body matches configured regexp", res.StatusCode, string(rawBody)))
				return nil
			}

			return fmt.Errorf("request result not successful with code: %v, body: %v", res.StatusCode, string(rawBody))
		}
	})

	return rawResponse, headers, statusCode, err
}

func (p *Http) unpackQueryValues(e *core.Event) (url.Values, error) {
	values := make(url.Values)

	for k, v := range p.Paramfields {
		field, err := e.GetField(v)
		if err != nil {
			continue // skip if target field not found
		}

		switch f := field.(type) {
		case []any:
			for i, j := range f {
				param, err := convert.AnyToString(j)
				if err != nil {
					return nil, fmt.Errorf("%v.%v: %w", v, i, err)
				}

				values.Add(k, param)
			}
		default:
			param, err := convert.AnyToString(f)
			if err != nil {
				return nil, fmt.Errorf("%v: %w", v, err)
			}

			values.Add(k, param)
		}
	}

	return values, nil
}

func (p *Http) requestMethod(e *core.Event) string {
	if l := p.MethodLabel; len(l) > 0 {
		if method, ok := e.GetLabel(l); ok && len(method) > 0 {
			return method
		}
		p.Log.Warn("event has no label with request method or it's empty")
	}
	return p.Method
}

func init() {
	plugins.AddProcessor("http", func() core.Processor {
		return &Http{
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
		}
	})
}
