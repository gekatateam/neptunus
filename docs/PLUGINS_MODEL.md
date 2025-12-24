# Neptunus Plugins Model

This section is for developers who want to create a new plugin.

There a nine types of plugins and some of it works directly with channels (we call it streaming plugins), and some of it not (callable or child plugins). Start by looking at the interfaces that plugins must conform to and some base structs that must be embedded - [here](../core/plugin.go) and [here](../core/base.go).

## Plugin lifecycle

Any plugin lifecycle is `create` -> `init` -> `set channels (for streaming plugins)` -> `run/call` -> `close`, and the engine takes care of some stages.

### Create

So, at the creation stage engine do some work:
 - initialize and set embedded base plugin (already with logger, metric observer and other fields, except channels);
 - if required, set child plugins, already created and initialized;
 - decode configuration to plugin struct.

Configuration decoder uses old good [mapstructure](https://github.com/mitchellh/mapstructure) with custom decode hooks - for time, duration and [datasize](https://pkg.go.dev/kythe.io/kythe/go/util/datasize#Size). See [kafka](../plugins/inputs/kafka/) as an example of datasize usage.

### Init and set channels

If creation stage completed successfully, engine call `Init() error` plugin method - and it is the place where your plugin MUST create and check all needed resources (e.g. database connection) and validate provided config. If something goes wrong, you MUST free all resources and return error. If no error returned, plugin is considered ready to data processing.

After that, the engine will create and set up channels. In most cases you don't need to think about it - base plugin handles `SetChannels(....)` call.

### Run

Then, if it is a streaming plugin, engine call `Run()` method and this method MUST be blocking. If it is a callable plugin, it will be called by it parent.

Inside `Run()` loop:
 - if your plugin is an `input`, you need to write events into `Out` channel;
 - if it is a `filter`, plugin read incoming events from `In` channel, do some calculations, then write it to `Acc` if condition satisfied, or write to `Rej` if it's not;
 - if it is a `processor`, plugin read incoming events from `In`, do some work, then write it to `Out` or to `Drop`, if event no longer needed;
 - finally, if it is an `output`, you read events from `In`, write it to target, and when you done with it, write event into `Done` channel.

So, the basic rule here is **any event MUST be sent to some channel in the end** because of [delivery control](DATA_MODEL.md#delivery-control).

Also, it is your responsibility to write plugin metrics. The base struct contains `Observe()` func which accepts status (Accepted, Failed or Rejected - the last one should be used in filters only) and duration taken to process the event.

You can find some heplers in [plugins/common/](../plugins/common/) dir, such as [Batcher](../plugins/common/batcher/), [Retryer](../plugins/common/retryer/) or [Pool](../plugins/common/pool/).

### Close

When the engine receives a signal to stop a pipeline, it calls inputs `Close() error` method.

If your plugin is an `input`, you need to handle this call, free all resources and break the `Run()` loop. Do not close the output channel! Engine will do this automatically.

If your plugin is a `filter`, `processor` or `output`, you must break the loop when plugin input channel closes. After that, the engine will call the `Close() error` method itself.

There is no guarantee that the close method will be called exactly once, so it MUST be idempotent.
