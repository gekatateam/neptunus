# Neptunus Plugins Model

This section is for developers who want to create a new plugin.

There are ten types of plugins. Some of them work directly with channels (we call them streaming plugins), while others do not (callable or child plugins). Start by looking at the interfaces that plugins must implement and the base structs that must be embedded: [here](../core/plugin.go) and [here](../core/base.go).

## Registration

Every plugin MUST be registered using one of the `Add` functions from the [registry](../plugins/registry.go), and its name MUST be unique within its type. You can do this in your package's `init()` function. This is also where you can set default values for your plugin configuration:
```go
func init() {
	plugins.AddInput("kafka", func() core.Input {
		return &Kafka{
			ClientId:          "neptunus.kafka.input",
			GroupId:           "neptunus.kafka.input",
			GroupBalancer:     "range",
			StartOffset:       "last",
			OnParserError:     "drop",
			GroupTTL:          24 * time.Hour,
			DialTimeout:       5 * time.Second,
			SessionTimeout:    30 * time.Second,
			RebalanceTimeout:  30 * time.Second,
			HeartbeatInterval: 3 * time.Second,
			ReadBatchTimeout:  3 * time.Second,
			WaitBatchTimeout:  3 * time.Second,
			MaxUncommitted:    100,
			CommitInterval:    1 * time.Second,
			MaxBatchSize:      datasize.Mebibyte, // 1 MiB,
			SASL: SASL{
				Mechanism: "none",
			},
			Ider:            &ider.Ider{},
			TLSClientConfig: &tls.TLSClientConfig{},
		}
	})
}
```

## Plugin lifecycle

The lifecycle of any plugin is `create` -> `init` -> `set channels (for streaming plugins)` -> `run/call` -> `stop (for streaming plugins)` -> `close`. The engine takes care of some of these stages.

### Create

At the creation stage, the engine does the following:
 - initializes and sets the embedded base plugin (including the logger, metric observer, and other fields, except channels);
 - if required, sets child plugins that have already been created and initialized;
 - decodes the configuration into the plugin struct.

The configuration decoder uses the [mapstructure](https://github.com/go-viper/mapstructure/v2) library with [custom decode hooks](../pkg/mapstructure/decoder.go). See [kafka](../plugins/inputs/kafka/) as an example of using datasize.

### Init and set channels

If the creation stage completes successfully, the engine calls the plugin's `Init() error` method. This is where your plugin MUST create and check all required resources (e.g. a database connection) and validate the provided configuration. If something goes wrong, you MUST free all resources and return an error. If no error is returned, the plugin is considered ready to work.

After that, the engine creates and sets up the channels. In most cases, you do not need to think about this: the base plugin handles the `SetChannels(....)` call.

### Run

Then, if it is a streaming plugin, the engine calls the `Run()` method, which MUST be blocking. If it is a callable plugin, it is called by its parent.

Inside `Run()` loop:
 - if your plugin is an `input`, you need to write events to the `Out` channel;
 - if it is a `filter`, the plugin reads incoming events from the `In` channel, performs any necessary calculations, and then writes the event to `Acc` if the condition is satisfied, or to `Rej` otherwise;
 - if it is a `processor`, the plugin reads incoming events from `In`, performs any necessary work, and then writes the event to `Out` or to `Drop` if the event is no longer needed;
 - finally, if it is an `output`, you read events from `In`, write them to the target, and when you are done with each event, write it to the `Done` channel.

So, the basic rule here is **any event MUST be sent to some channel in the end** because of [delivery control](DATA_MODEL.md#delivery-control).

It is also your responsibility to write plugin metrics. The base struct contains an `Observe()` function that accepts a status (Accepted, Failed, or Rejected - the last one should be used only in filters) and the time taken to process the event.

You can find some helpers in the [plugins/common/](../plugins/common/) directory, such as [Batcher](../plugins/common/batcher/), [Retryer](../plugins/common/retryer/), and [Pool](../plugins/common/pool/).

For callable plugins, please remember that a plugin may be called simultaneously from multiple goroutines, so make it concurrency-safe.

### Stop

When the engine receives a signal to stop a pipeline, it calls the inputs' `Stop()` method.

If your plugin is an `input`, you need to handle this call, stop consuming events, and break the `Run()` loop. Do not close the output channel. The engine will do this automatically.

If your plugin is a `filter`, `processor`, or `output`, you must break the loop when the plugin's input channel closes.

### Close

When the pipeline has fully stopped, the engine calls the plugins' `Close() error` method. This is where you MUST free all resources.

There is no guarantee that the close method will be called exactly once, so it MUST be idempotent.

If your streaming plugin uses callable plugins, you need to close them in `Close() error`.

## Lookups and Keykeepers

There are two specific types of plugins with similar functionality but different purposes:
 - `keykeepers` - used only at pipeline startup to obtain configuration from external sources;
 - `lookups` - always running in the background to obtain data at runtime.

The keykeeper lifecycle is identical to that of callable plugins. After successful initialization, the pipeline calls the `Get(key string) (any, error)` method. The key format is specified by the keykeeper. For examples, see [vault](../plugins/keykeepers/vault/) or [env](../plugins/keykeepers/env/).

On the other hand, a lookup is closer to a streaming plugin. Like an `input`, it has a `Run()` loop and a `Stop()` method. Inside this loop, the lookup updates stored data and provides it through the `Get(key string) (any, error)` call. The key format here is always a dot-separated key path, as in the [event fields API](./DATA_MODEL.md).

The lookup's `Get` method may be called simultaneously from multiple goroutines, so make it concurrency-safe with respect to internal data updates. The `Get` method MUST also return a copy of the data.

If you want to create a simple lookup without specific update logic, you can wrap it with the [core plugin](../plugins/core/lookup/lookup.go), which performs an update loop at the configured interval and protects the data with a read-write mutex. See [file lookup](../plugins/lookups/file/) as an example.
