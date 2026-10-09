# Ollama Processor Plugin

The `ollama` processor plugin sends a generation request to an Ollama API for each event. It can read the prompt from an event field and write the generated response to an event field. Requests are sent without streaming.

# Configuration
```toml
[[processors]]
  [processors.ollama]
    # Ollama API URL, required
    host = "http://localhost:11434"

    # bearer token for API authorization, required
    token = "@{envs:OLLAMA_TOKEN}"

    # model name, required
    model = "llama3.2"

    # system prompt sent with each generation request
    system_prompt = "You are a helpful assistant."

    # event field containing the prompt
    # if empty, an empty prompt is sent
    prompt_from = "prompt"

    # event field to receive the generated response
    # if empty, the response is not added to the event
    response_to = "ollama.response"

    # response format; use "json" to request JSON output
    # or a JSON schema object as string
    format = ""

    # send the prompt as-is, without normal prompt templating
    raw = false

    # duration to keep the model loaded after the request
    keep_alive = "5m"

    # timeout for the HTTP client and generation request
    timeout = "60s"

    # maximum time an idle connection remains open
    idle_conn_timeout = "1m"

    # maximum number of idle connections
    max_idle_conns = 10

    # maximum number of generation attempts
    # zero means retry indefinitely
    retry_attempts = 0

    # delay between generation retries
    retry_after = "5s"

    ## TLS configuration
    # if true, TLS client will be used
    tls_enable = false
    # trusted root certificates for server
    tls_ca_file = "/etc/neptunus/ca.pem"
    # used for TLS client certificate authentication
    tls_key_file = "/etc/neptunus/key.pem"
    tls_cert_file = "/etc/neptunus/cert.pem"
    # minimum TLS version, not limited by default
    tls_min_version = "TLS12"
    # send the specified TLS server name via SNI
    tls_server_name = "ollama.local"
    # use TLS but skip chain & host verification
    tls_insecure_skip_verify = false

    # Ollama generation options, for example temperature or num_ctx
    # see API reference - https://docs.ollama.com/api/generate
    # and used package struct - https://github.com/prathyushnallamothu/ollamago/blob/master/types.go#L14
    [processors.ollama.options]
      temperature = 0.7
      num_ctx = 4096
```
