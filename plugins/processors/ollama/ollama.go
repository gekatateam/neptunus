package ollama

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
	"github.com/tmc/langchaingo/llms/ollama"
	"golang.org/x/exp/slog"
)

// Ollama processor integrates with Ollama for language model processing
type Ollama struct {
	*core.BaseProcessor `mapstructure:"-"`
	// BaseURL for the Ollama API
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
	// ollama client
	client llms.LLM
}

// Start begins the Ollama processor
func (p *Ollama) Init() error {
	if p.Model == "" {
		return fmt.Errorf("model is required")
	}

	if p.PromptFrom == "" {
		return fmt.Errorf("prompt_from field is required")
	}

	if p.ResponseTo == "" {
		return fmt.Errorf("response_to field is required")
	}

	// Set defaults
	if p.BaseURL == "" {
		p.BaseURL = "http://localhost:11434"
	}

	if p.TimeoutSeconds == 0 {
		p.TimeoutSeconds = 30
	}

	if p.Temperature == 0 {
		p.Temperature = 0.7
	}

	// Initialize Ollama client
	client, err := ollama.New(
		ollama.WithServerURL(p.BaseURL),
		ollama.WithModel(p.Model),
		ollama.WithKeepAlive("5m"),
		ollama.WithRunnerNumCtx(20000),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize Ollama client: %w", err)
	}
	p.client = client

	return nil
}

// Stop the processor
func (p *Ollama) Close() error {
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
func (p *Ollama) Run() {
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
		//opts = append(opts, llms.WithJSONMode())

		content := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, p.SystemPrompt),
			llms.TextParts(llms.ChatMessageTypeHuman, prompt),
		}

		// Call Ollama
		completion, err := p.client.GenerateContent(ctx, content,
			opts...,
		)
		cancel()
		if err != nil {
			p.handleError(e, now, fmt.Errorf("Ollama API call failed: %w", err))
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
func (p *Ollama) handleError(e *core.Event, startTime time.Time, err error) {
	p.Log.Error("Ollama processor error",
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
func (p *Ollama) Observe(status metrics.EventStatus, duration time.Duration) {
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
	plugins.AddProcessor("ollama", func() core.Processor {
		return &Ollama{
			Temperature:    0.7,
			TimeoutSeconds: 30,
		}
	})
}
