# Neptunus Plugins Model

This section is for developers who want to create a new plugin.

There a nine types of plugins and some of it works directly with channels (we call it streaming plugins), and some of it not (callable or child plugins). Start by looking at the interfaces that plugins must conform to and some base structs that must be embedded - [here](../core/plugin.go) and [here](../core/base.go).

Any plugin lifecycle is `create` -> `init` -> `set channels (for streaming plugins)` -> `run/call` -> `close`, and the engine takes care of some stages.

So, at the creation stage, engine do some work:
 - initialize and set embedded base plugin (already with logger, metric observer and other fields, except channels);
 - if required, set child plugins, already created and initialized;
 - decode configuration to plugin struct.

Configuration decoder uses old good [mapstructure](https://github.com/mitchellh/mapstructure) with custom decode hooks - for time, duration and [datasize](https://pkg.go.dev/kythe.io/kythe/go/util/datasize#Size). See [kafka](../plugins/inputs/kafka/) as an example of datasize usage.

If creation stage completed successfully, engine call `Init() error` plugin method - and it is the place, where your plugin MUST create and check all needed resources (e.g. database connection) and validate provided config. If something goes wrong, you MUST free all resources and return error. If no error returned, plugin is considered ready to data processing.

After that, the engine will create and set up channels. In most cases you don't need to think about it - base plugin handles `SetChannels(....)` call.

Then, if it is a streaming plugin
