# Template Text Serializer Plugin
The `template_text` serializer plugin converts events to text using [Go templates](https://pkg.go.dev/text/template). [Slim-sprig functions](https://go-task.github.io/slim-sprig/) are also available.

The plugin uses [wrapped events](../../common/template/README.md).

# Configuration
```toml
[[outputs]]
  [outputs.log]
    [outputs.log.serializer]
      type = "template_text"

      # string with template
      template_text = '''
{{- range $event := . -}}
Alert: {{ $event.GetField "alert_name" }}. Ok result: {{ $event.GetField "ok_result" }}. Err result: {{ $event.GetField "err_result" }}
{{- end -}}
'''

      # path to template file
      # takes precedence over "template_text" parameter
      template_path = "path/to/template.tmpl"
```
