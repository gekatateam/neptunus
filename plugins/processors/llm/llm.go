package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
	"golang.org/x/exp/slog"
)

// LLM processor integrates with LLM for language model processing
type LLM struct {
	*core.BaseProcessor `mapstructure:"-"`
	// Engine LLM to use, support ollama, google, openai
	LLMType string `mapstructure:"llm_type"`
	// BaseURL for the LLM API
	BaseURL string `mapstructure:"base_url"`
	// Model to use for generation
	Model string `mapstructure:"model"`
	// PromptFrom field name to get prompt from
	PromptFrom string `mapstructure:"prompt_from"`
	// ResponseTo field name to store response
	ResponseTo string `mapstructure:"response_to"`
	// Temperature controls randomness (0.0-1.0)
	Temperature float64 `mapstructure:"temperature"`
	// MaxTokens maximum number of tokens to generate
	MaxTokens int `mapstructure:"max_tokens"`
	// TimeoutSeconds for request
	TimeoutSeconds int `mapstructure:"timeout_seconds"`
	// SystemPrompt to use as context
	SystemPrompt string `mapstructure:"system_prompt"`
	// JSONMode to enable JSON output
	JSONMode bool `mapstructure:"json_mode"`
	// KeepAlive controls how long the model will stay loaded into memory following the request
	KeepAlive string `mapstructure:"keep_alive"`
	// api_key for llm
	ApiKey string `mapstructure:"api_key"`
	// ollama client
	client llms.Model
}

// Start begins the LLM processor
func (p *LLM) Init() error {
	if p.LLMType == "" {
		return fmt.Errorf("llm_type is required, support ollama, google, openai")
	}
	if p.Model == "" {
		return fmt.Errorf("model is required")
	}

	if p.PromptFrom == "" {
		return fmt.Errorf("prompt_from field is required")
	}

	if p.ResponseTo == "" {
		return fmt.Errorf("response_to field is required")
	}

	if p.SystemPrompt == "" {
		return fmt.Errorf("system_prompt field is required")
	}

	if p.MaxTokens == 0 {
		p.MaxTokens = 4096
	}

	// Set defaults
	if p.BaseURL == "" {
		p.BaseURL = "http://localhost:11434"
	}

	if p.TimeoutSeconds == 0 {
		p.TimeoutSeconds = 30
	}

	if p.Temperature == 0 {
		p.Temperature = 0.5
	}

	if p.KeepAlive == "" {
		p.KeepAlive = "5m"
	}

	// Initialize LLM client
	var client llms.Model
	var err error
	switch p.LLMType {
	case "ollama":
		client, err = ollama.New(
			ollama.WithServerURL(p.BaseURL),
			ollama.WithModel(p.Model),
			ollama.WithKeepAlive(p.KeepAlive),
			ollama.WithRunnerNumCtx(p.MaxTokens),
		)
		break
	case "google":
		client, err = googleai.New(context.Background(),
			googleai.WithAPIKey(p.ApiKey),
			googleai.WithDefaultModel(p.Model),
			googleai.WithDefaultMaxTokens(p.MaxTokens),
			googleai.WithDefaultTemperature(p.Temperature),
			googleai.WithDefaultEmbeddingModel("embedding-001"),
			googleai.WithDefaultCandidateCount(1),
			googleai.WithDefaultTopK(3),
			googleai.WithDefaultTopP(0.95),
			googleai.WithHarmThreshold(googleai.HarmBlockNone),
		)
		break
	case "openai":
		client, err = openai.New(
			openai.WithBaseURL(p.BaseURL),
			openai.WithModel(p.Model),
			openai.WithToken(p.ApiKey),
		)
	}
	if err != nil {
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}
	p.client = client

	return nil
}

// Stop the processor
func (p *LLM) Close() error {
	return nil
}

func FlattenMap(m map[string]interface{}) map[string]interface{} {
	flat := make(map[string]interface{})
	for k, v := range m {
		if vm, ok := v.(map[string]interface{}); ok {
			for kk, vv := range FlattenMap(vm) {
				flat[k+"."+kk] = vv
			}
		} else {
			flat[k] = v
		}
	}
	return flat
}

func GetJSONData(s string) (map[string]interface{}, error) {
	data := strings.TrimPrefix(s, "```json")
	data = strings.TrimSuffix(data, "```")
	var i map[string]interface{}
	err := json.Unmarshal([]byte(data), &i)
	if err != nil {
		return nil, err
	}
	return i, nil
}

// Process events from input channel
func (p *LLM) Run() {
	for e := range p.In {
		now := time.Now()

		// Get prompt from event
		fieldValue, err := e.GetField(p.PromptFrom)
		if err != nil {
			p.handleError(e, now, fmt.Errorf("failed to get prompt field '%s': %w", p.PromptFrom, err))
			continue
		}

		prompt, err := toString(fieldValue)
		if err != nil {
			p.handleError(e, now, fmt.Errorf("failed to convert prompt field to string: %w", err))
			continue
		}

		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.TimeoutSeconds)*time.Second)

		// Prepare generation options
		opts := []llms.CallOption{}
		if p.MaxTokens > 0 {
			opts = append(opts, llms.WithMaxTokens(p.MaxTokens))
		}
		if p.JSONMode {
			opts = append(opts, llms.WithJSONMode())
		}

		content := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, p.SystemPrompt),
			llms.TextParts(llms.ChatMessageTypeHuman, prompt),
		}

		// Call LLM
		completion, err := p.client.GenerateContent(ctx, content,
			opts...,
		)
		cancel()
		if err != nil {
			p.handleError(e, now, fmt.Errorf("LLM API call failed: %w", err))
			continue
		}

		if completion == nil || len(completion.Choices) == 0 {
			p.handleError(e, now, fmt.Errorf("no completion data"))
			continue
		}

		if p.JSONMode {
			it, err := GetJSONData(completion.Choices[0].Content)
			if err != nil {
				p.handleError(e, now, fmt.Errorf("failed to parse JSON response: %w", err))
				continue
			}
			it = FlattenMap(it)
			for k, v := range it {
				if err := e.SetField(k, v); err != nil {
					p.handleError(e, now, fmt.Errorf("failed to set response field '%s': %w", k, err))
					continue
				}
			}
		} else {
			// Set response in event
			if err := e.SetField(p.ResponseTo, completion.Choices[0].Content); err != nil {
				p.handleError(e, now, fmt.Errorf("failed to set response field '%s': %w", p.ResponseTo, err))
				continue
			}
		}

		e.SetLabel("SystemPrompt", p.SystemPrompt)
		e.SetLabel("UserPrompt", prompt)
		e.SetLabel("Response", completion.Choices[0].Content)
		info, err := toString(completion.Choices[0].GenerationInfo)
		if err == nil {
			e.SetLabel("GenerationInfo", info)
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

// handleError processes errors in a uniform way
func (p *LLM) handleError(e *core.Event, startTime time.Time, err error) {
	p.Log.Error("LLM processor error",
		"error", err,
		slog.Group("event",
			"id", e.Id,
			"key", e.RoutingKey,
		),
	)
	e.StackError(err)
	p.Out <- e
	p.Observe(metrics.EventFailed, time.Since(startTime))
}

// Observe metrics
func (p *LLM) Observe(status metrics.EventStatus, duration time.Duration) {
	// Implement metrics observation based on your metrics package
	// This is a placeholder
}

// Helper function to convert any value to string
func toString(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case fmt.Stringer:
		return v.String(), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// Register the processor
func init() {
	plugins.AddProcessor("llm", func() core.Processor {
		return &LLM{
			TimeoutSeconds: 30,
		}
	})
}
