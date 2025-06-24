# Template Processor Plugin

The `template` processor uses [Golang templates](https://pkg.go.dev/text/template) to modify/create event Id, routing key, labels and fields. [Slim-sprig functions](https://go-task.github.io/slim-sprig/) available!

Plugin uses [wrapped events](../../common/template/README.md).

If template execution or field setting fails, event is marked as failed, but other templates execution continues.

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
