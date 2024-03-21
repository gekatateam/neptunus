# Parser Processor Plugin

The `parser` processor parses string or bytes slice field into (new) event(s). This plugin requires a parser in it's configuration.

## Configuration
```toml
[[processors]]
  [processors.parser]
    # parser mode, "merge" or "produce"
    # in produce mode, plugin generates new events from parser result
    # origin labels, tags, id and routing key will be copied into new events
    #
    # in merge mode, plugin merges parser result with input event
    # if parser returns multiple events, plugin produces 
    # as many events as the parser returned (see example)
    behaviour = "merge"

    # path to field, the contents of which 
    # will be passed to a parser
    # must be string or bytes slice
    from = "path.to.field"

    # if true
    # in produce mode, plugin will drop origin event
    # in merge mode, plugin will drop "from" field
    drop_origin = true

    # only using in merge mode
    # path to field, where result of a parser's work will be placed
    # if empty, fields will be placed in root of event data map
    to = "path.to.field"

    # only using in produce mode
    # if configured, an event id will be set by data from path
    # expected format - "type:path"
    id_from = "field:path.to.id"
    [processors.parser.parser]
      type = "json"
```

## Examples
A little more about merge mode.

This is how plugin works, when parser returns multiple events (if `to` is empty):
```json
# input event data
{
    "level": "info",
    "message": "[{\"log\":\"Log line 11 here\",\"stream\":\"stdout\",\"time\":\"2019-01-01T11:11:11.111111111Z\"}, {\"log\":\"Log line 22 here\",\"stream\":\"stdout\",\"time\":\"2019-01-01T11:11:11.111111111Z\"}]",
}

# output events data
{
    "level": "info",
    "message": "[{\"log\":\"Log line 11 here\",\"stream\":\"stdout\",\"time\":\"2019-01-01T11:11:11.111111111Z\"}, {\"log\":\"Log line 22 here\",\"stream\":\"stdout\",\"time\":\"2019-01-01T11:11:11.111111111Z\"}]",
    "log":"Log line 11 here",
    "stream":"stdout",
    "time":"2019-01-01T11:11:11.111111111Z"
}
{
    "level": "info",
    "message": "[{\"log\":\"Log line 11 here\",\"stream\":\"stdout\",\"time\":\"2019-01-01T11:11:11.111111111Z\"}, {\"log\":\"Log line 22 here\",\"stream\":\"stdout\",\"time\":\"2019-01-01T11:11:11.111111111Z\"}]",
    "log":"Log line 22 here",
    "stream":"stdout",
    "time":"2019-01-01T11:11:11.111111111Z"
}
```
