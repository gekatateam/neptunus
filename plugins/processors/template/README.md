# Template Processor Plugin

The `template` processor uses [Go templates](https://pkg.go.dev/text/template) to modify or create the event ID, routing key, labels, and fields. [Slim-sprig functions](https://go-task.github.io/slim-sprig/) are available.

The plugin uses [wrapped events](../../common/template/README.md).

If template execution or field setting fails, the event is marked as failed, but execution continues for the other templates.

## Configuration
```toml
[[processors]]
  [processors.template]
    # routing key template
    routing_key = '{{ .RoutingKey }}-{{ .Timestamp.Format "2006-01-02" }}'

    # id template
    id = '{{ .GetLabel "message_id" }}'

    # "label name <- template" map
    [processors.template.labels]
      host = '{{ .GetField "client.host" }}:{{ .GetField "client.port" }}'

    # "field path <- template" map
    [processors.template.fields]
      "metadata.full_address" = '{{ .GetField "address.street" }}, {{ .GetField "address.building" }}'
```
