# Template Serializer Plugin
The `template_text` serializer plugin converts events into text via [template](https://pkg.go.dev/text/template).

# Configuration
```toml
[[outputs]]
  [outputs.log]
  [outputs.log.serializer]
    type = "template_text"
    
    # string with template
    template_text = "Hello, {{ .Name }}!"
    
    # path to template file
    # takes precedence over "template_text" variable
    template_path = "path/to/template_file"
```