# LLM Processor Plugin

The `llm` processor plugin integrates with Large Language Models (LLMs) to process event data. It supports multiple LLM providers including Ollama, Google AI, and OpenAI.

This processor allows you to send prompts to language models and capture their responses as part of your event processing pipeline. It can optionally save all interactions to CSV files for auditing and analysis.

## Configuration

```toml
[[processors]]
  [processors.llm]
    # LLM engine to use, required
    # supports: "ollama", "google", "openai"
    llm_type = "ollama"

    # Base URL for the LLM API
    # Default: "http://localhost:11434" for Ollama
    base_url = "http://localhost:11434"

    # Model to use for generation, required
    model = "llama2"

    # Field name to get prompt from, required
    prompt_from = "message.content"

    # Field name to store response
    # Default: "llm.output"
    response_to = "message.response"

    # Controls randomness (0.0-1.0)
    # Default: 0.5
    temperature = 0.7

    # Maximum number of tokens to generate
    # Default: 4096
    max_tokens = 2048

    # Timeout for request in seconds
    # Default: 30
    timeout_seconds = 60

    # System prompt to use as context, required
    system_prompt = "You are a helpful assistant."

    # Enable JSON mode for structured output
    # Default: false
    json_mode = false

    # Extract a specific key from JSON output
    # Only used when json_mode is true
    json_mode_get_key = "result"

    # Controls how long the model stays loaded in memory (Ollama)
    # Default: "5m"
    keep_alive = "10m"

    # API key for the LLM provider (required for OpenAI and Google)
    api_key = "your-api-key"

    # Path to save CSV records of all interactions
    # If not set, interactions are not saved
    save_csv_path = "/path/to/llm_interactions.csv"

    # Maximum file size for CSV before rotation (in bytes)
    # Default: 50MB (52,428,800 bytes)
    max_csv_size = 52428800
```

## Providers

### Ollama

For Ollama, set `llm_type = "ollama"` and ensure Ollama is running at the specified `base_url`. The default configuration assumes Ollama is running locally.

### Google AI

For Google AI, set `llm_type = "google"` and provide your API key in the `api_key` field.

### OpenAI

For OpenAI, set `llm_type = "openai"`, provide your API key in the `api_key` field, and optionally set a custom `base_url` if using a proxy or compatible API.

## JSON Mode

When `json_mode` is enabled, the processor will request structured JSON output from the LLM. The response will be processed in one of two ways:

1. If `json_mode_get_key` is specified, the processor will:
   - Extract the value of the specified key from the JSON response
   - Set this value to the `response_to` field

2. If `json_mode_get_key` is not specified, the processor will:
   - Set the entire JSON response (without the markdown code block delimiters) to the `response_to` field

## CSV Logging

When `save_csv_path` is configured, the processor logs all interactions to a CSV file with the following columns:
- timestamp
- model
- system_prompt
- prompt
- response
- llm_type

The CSV file automatically rotates when its size exceeds `max_csv_size`.

## Event Labels

The processor adds the following labels to the processed event:
- `SystemPrompt`: The system prompt used
- `UserPrompt`: The user prompt sent to the LLM
- `Response`: The LLM's response
- `GenerationInfo`: Additional generation metadata (when available)

## Example Usage

```toml
[[processors]]
  [processors.llm]
    llm_type = "ollama"
    model = "mistral"
    prompt_from = "user.question"
    response_to = "ai.answer"
    system_prompt = "You are an AI assistant that provides helpful, accurate, and concise responses."
    temperature = 0.3
    max_tokens = 1024
    save_csv_path = "/var/log/neptunus/llm_interactions.csv"
```

This configuration will:
1. Take the prompt from the `user.question` field
2. Send it to the Mistral model on the local Ollama server
3. Store the response in the `ai.answer` field
4. Log all interactions to the specified CSV file
