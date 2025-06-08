# Dynamic Template Processor Plugin

The `dynamic_template` processor evaluates [Golang templates](https://pkg.go.dev/text/template) in configured labels and fields. [Slim-sprig functions](https://go-task.github.io/slim-sprig/) available!

Unlike [template processor](../template/), this plugin does not use predefined templates.

Plugin uses [wrapped events](../../common/template/README.md).

If template execution fails, event is marked as failed, but other templates execution continues.

## Configuration
```toml
[[processors]]
  [processors.dynamic_template]
    # compiled template TTL
    template_ttl = "1h"

    # list of labels to evaluate
    labels = [ "hello_message" ]

    # list of fields to evaluate
    # field must be a string or a slice/map of strings
    fields = [ "annotations" ]
```
