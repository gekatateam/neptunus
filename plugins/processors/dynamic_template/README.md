# Dynamic Template Processor Plugin

The `dynamic_template` processor evaluates [Go templates](https://pkg.go.dev/text/template) in the configured labels and fields. [Slim-sprig functions](https://go-task.github.io/slim-sprig/) are available.

Unlike the [template processor](../template/), this plugin does not use predefined templates. Instead, it uses the **label or field content** as a template and replaces it with the execution result.

The plugin uses [wrapped events](../../common/template/README.md).

If template execution fails, the event is marked as failed, but execution continues for the other templates.

> [!TIP]  
> This plugin may write its own [metrics](../../../docs/METRICS.md#internal-caches)

## Configuration
```toml
[[processors]]
  [processors.dynamic_template]
    # if true, plugin metrics cache length exposed as metric
    enable_metrics = false

    # compiled template TTL
    template_ttl = "1h"

    # list of labels to evaluate
    labels = [ "hello_message" ]

    # list of fields to evaluate
    # field must be a string or a slice/map of strings
    fields = [ "annotations" ]
```
