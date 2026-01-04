# Neptunus Plugins Model

This section is for developers who want to create a new plugin.

There a nine types of plugins and some of it works directly with channels (we call it streaming plugins), and some of it not (callable or child plugins). Start by looking at the interfaces that plugins must conform to and some base structs that must be embedded - [here](../core/plugin.go) and [here](../core/base.go).

## Registration

Any plugin MUST be registered using one of the `Add` functions from [registry](../plugins/registry.go) and it's name MUST be unique across it's type. You can do it in your package `init()` func. Also, it is the place where you can set default values for your plugin configuration:
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

Any plugin lifecycle is `create` -> `init` -> `set channels (for streaming plugins)` -> `run/call` -> `stop (for streaming plugins)` -> `close`, and the engine takes care of some stages.

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

In case of callable plugins, please remember that a plugin may be called simultaneously from multiple goroutines, so, make it concurrent-safety.

### Stop
When the engine receives a signal to stop a pipeline, it calls inputs `Stop()` method.

If your plugin is an `input`, you need to handle this call, stop consuming events and break the `Run()` loop. Do not close the output channel! Engine will do this automatically.

If your plugin is a `filter`, `processor` or `output`, you must break the loop when plugin input channel closes.

### Close

When pipeline fully stopped, engine calls plugins `Close() error` method - and it the place where you MUST free all resources. 

There is no guarantee that the close method will be called exactly once, so it MUST be idempotent.

If your streaming plugin uses some callable plugins, you need to close it in `Close() error`.
