package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/convert"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

type Http struct {
	*core.BaseProcessor `mapstructure:"-"`
	Host                string        `mapstructure:"host"`
	Method              string        `mapstructure:"method"`
	Timeout             time.Duration `mapstructure:"timeout"`
	IdleConnTimeout     time.Duration `mapstructure:"idle_conn_timeout"`
	MaxIdleConns        int           `mapstructure:"max_idle_conns"`
	SuccessCodes        []int         `mapstructure:"success_codes"`
	PathLabel           string        `mapstructure:"path_label"`

	Headerlabels map[string]string `mapstructure:"headerlabels"`
	Paramfields  map[string]string `mapstructure:"paramfields"`

	RequestBodyFrom string `mapstructure:"request_body_from"`
	ResponseBodyTo  string `mapstructure:"response_body_to"`

	*tls.TLSClientConfig `mapstructure:",squash"`
	*retryer.Retryer     `mapstructure:",squash"`

	successCodes map[int]struct{}
	baseUrl      *url.URL
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

	url, err := url.ParseRequestURI(p.Host)
	if err != nil {
		return err
	}
	p.baseUrl = url

	successCodes := map[int]struct{}{}
	for _, v := range p.SuccessCodes {
		successCodes[v] = struct{}{}
	}
	p.successCodes = successCodes

	tlsConfig, err := p.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	p.client = clientStorage.CompareAndStore(p.id, &http.Client{
		Timeout: p.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
			IdleConnTimeout: p.IdleConnTimeout,
			MaxIdleConns:    p.MaxIdleConns,
		},
	})

	return nil
}

func (p *Http) Close() error {
	p.ser.Close()
	p.parser.Close()
	p.client.CloseIdleConnections()
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
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
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
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
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
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
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
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				e.StackError(fmt.Errorf("event does not contains %v label", p.PathLabel))
				p.Out <- e
				p.Observe(metrics.EventFailed, time.Since(now))
				continue
			}
			path = label
		}

		respBody, err := p.perform(p.baseUrl.JoinPath(path), values, reqBody, header)
		if err != nil {
			p.Log.Error("request failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.StackError(err)
			p.Out <- e
			p.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		if len(respBody) > 0 && len(p.ResponseBodyTo) > 0 {
			events, err := p.parser.Parse(respBody, "")
			if err != nil {
				p.Log.Error("response processing failed",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
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
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				e.StackError(fmt.Errorf("error set response data: %w", err))
				p.Out <- e
				p.Observe(metrics.EventFailed, time.Since(now))
				continue
			}
		}

		p.Log.Debug("event processed",
			slog.Group("event",
				"id", e.Id,
				"key", e.RoutingKey,
			),
		)
		p.Out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func (p *Http) perform(uri *url.URL, params url.Values, body []byte, header http.Header) ([]byte, error) {
	uri.RawQuery = params.Encode()

	var rawResponse []byte

	err := p.Retryer.Do("perform request", p.Log, func() error {
		req, err := http.NewRequest(p.Method, uri.String(), bytes.NewReader(bytes.Clone(body)))
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

		if _, ok := p.successCodes[res.StatusCode]; ok {
			p.Log.Debug(fmt.Sprintf("request performed successfully with code: %v, body: %v", res.StatusCode, string(rawBody)))
			return nil
		} else {
			return fmt.Errorf("request result not successfull with code: %v, body: %v", res.StatusCode, string(rawBody))
		}
	})

	return rawResponse, err
}

func (r *Http) unpackQueryValues(e *core.Event) (url.Values, error) {
	values := make(url.Values)

	for k, v := range r.Paramfields {
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
