# Template Common Plugin

This plugin provides a wrapper around an event for use in [Go templates](https://pkg.go.dev/text/template).

TEvent methods:
 - `RoutingKey() string`
 - `Id() string`
 - `Timestamp() time.Time`
 - `Errors() []string` - returns event errors
 - `GetLabel(key string) string` - returns empty string if an event has no label associated with key
 - `GetField(path string) any` - returns `nil` if an event has no field on passed path
