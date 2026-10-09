package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/prathyushnallamothu/ollamago"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/elog"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
	"github.com/gekatateam/neptunus/plugins/common/sharedstorage"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

var clientStorage = sharedstorage.New[*ollamago.Client, uint64]()

type Ollama struct {
	*core.BaseProcessor  `mapstructure:"-"`
	Host                 *url.URL       `mapstructure:"host"`
	Token                string         `mapstructure:"token"`
	Model                string         `mapstructure:"model"`
	SystemPrompt         string         `mapstructure:"system_prompt"`
	PromptFrom           string         `mapstructure:"prompt_from"`
	ResponseTo           string         `mapstructure:"response_to"`
	Format               string         `mapstructure:"format"`
	Raw                  bool           `mapstructure:"raw"`
	KeepAlive            time.Duration  `mapstructure:"keep_alive"`
	Options              map[string]any `mapstructure:"options"`
	Timeout              time.Duration  `mapstructure:"timeout"`
	IdleConnTimeout      time.Duration  `mapstructure:"idle_conn_timeout"`
	MaxIdleConns         int            `mapstructure:"max_idle_conns"`
	*tls.TLSClientConfig `mapstructure:",squash"`
	*retryer.Retryer     `mapstructure:",squash"`

	format  json.RawMessage
	options *ollamago.Options // https://github.com/prathyushnallamothu/ollamago/blob/master/types.go#L14
	client  *ollamago.Client
	id      uint64
}

func (p *Ollama) Init() error {
	if p.Host == nil {
		return errors.New("host required")
	}

	if len(p.Model) == 0 {
		return errors.New("model required")
	}

	if len(p.Token) == 0 {
		return errors.New("token required")
	}

	tlsConfig, err := p.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	p.client = clientStorage.CompareAndStore(p.id, ollamago.NewClient(
		ollamago.WithHTTPClient(&http.Client{
			Timeout: p.Timeout,
			Transport: &http.Transport{
				TLSClientConfig:   tlsConfig,
				IdleConnTimeout:   p.IdleConnTimeout,
				MaxIdleConns:      p.MaxIdleConns,
				ForceAttemptHTTP2: tlsConfig != nil,
			},
		}),
		ollamago.WithBaseURL(p.Host.String()),
		ollamago.WithHeader("Authorization", "Bearer "+p.Token),
	))

	p.options = &ollamago.Options{}
	dec, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  p.options,
	})

	if err := dec.Decode(p.Options); err != nil {
		return fmt.Errorf("failed to decode options: %w", err)
	}

	p.format = json.RawMessage(p.Format)
	if p.Format == "json" {
		p.format = json.RawMessage(`"json"`)
	}

	return nil
}

func (p *Ollama) SetId(id uint64) {
	p.id = id
}

func (p *Ollama) Close() error {
	clientStorage.Leave(p.id)
	return nil
}

func (p *Ollama) Run() {
	for e := range p.In {
		now := time.Now()

		prompt := ""
		if len(p.PromptFrom) > 0 {
			rawPrompt, err := e.GetField(p.PromptFrom)
			if err != nil {
				p.Log.Error("failed to get prompt from event",
					"error", err,
					elog.EventGroup(e),
				)
				e.StackError(err)
				p.Out <- e
				p.Observe(metrics.EventFailed, time.Since(now))
				continue
			}

			var ok bool
			prompt, ok = rawPrompt.(string)
			if !ok {
				p.Log.Error("failed to convert prompt to string",
					"error", err,
					elog.EventGroup(e),
				)
				e.StackError(err)
				p.Out <- e
				p.Observe(metrics.EventFailed, time.Since(now))
				continue
			}
		}

		req := ollamago.GenerateRequest{
			Model:     p.Model,
			System:    p.SystemPrompt,
			Format:    p.format,
			Raw:       p.Raw,
			KeepAlive: p.KeepAlive.String(),
			Prompt:    prompt,
			Stream:    false,
			Options:   p.options,
		}

		response, err := p.generate(req)
		if err != nil {
			p.Log.Error("failed to generate response",
				"error", err,
				elog.EventGroup(e),
			)
			e.StackError(err)
			p.Out <- e
			p.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		p.Log.Debug(fmt.Sprintf("full api call result: %#v", *response),
			elog.EventGroup(e),
		)

		if len(p.ResponseTo) > 0 {
			e.SetField(p.ResponseTo, response.Response)
		}

		p.Log.Debug("response generated",
			elog.EventGroup(e),
		)
		p.Out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func (p *Ollama) generate(req ollamago.GenerateRequest) (*ollamago.GenerateResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
	defer cancel()

	var response *ollamago.GenerateResponse
	err := p.Retryer.Do("generate response", p.Log, func() error {
		resp, err := p.client.Generate(ctx, req)
		if err != nil {
			return err
		}

		response = resp
		return nil
	})
	return response, err
}

func init() {
	plugins.AddProcessor("ollama", func() core.Processor {
		return &Ollama{
			Timeout:         60 * time.Second,
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
