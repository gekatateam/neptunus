# Neptunus metrics

## Pipelines common

Metrics that writes by each pipeline

#### Gauge `pipeline_state`
Pipeline state: 1-5 is for Created, Starting, Running, Stopping, Stopped.

Labels:
 - **pipeline** - pipeline Id

#### Gauge `pipeline_processors_lines`
Count of configured processors lines.

Labels:
 - **pipeline** - pipeline Id

## Plugins common

Metrics that writes by each plugin, depends on plugin kind

#### Summary `input_plugin_processed_events`
Events statistic for inputs.

Quantiles: 0.5, 0.9, 0.99, 1.0

Labels:
 - **pipeline** - pipeline Id
 - **plugin** - plugin type
 - **name** - plugin name (alias)
 - **status** - event status, `failed`, `rejected` or `accepted`

#### Summary `filter_plugin_processed_events`
Events statistic for filters.

Quantiles: 0.5, 0.9, 0.99, 1.0

Labels:
 - **pipeline** - pipeline Id
 - **plugin** - plugin type
 - **name** - plugin name (alias)
 - **status** - event status, `failed`, `rejected` or `accepted`

#### Summary `processor_plugin_processed_events`
Events statistic for processors.

Quantiles: 0.5, 0.9, 0.99, 1.0

Labels:
 - **pipeline** - pipeline Id
 - **plugin** - plugin type
 - **name** - plugin name (alias)
 - **status** - event status, `failed`, `rejected` or `accepted`

#### Summary `output_plugin_processed_events`
Events statistic for outputs.

Quantiles: 0.5, 0.9, 0.99, 1.0

Labels:
 - **pipeline** - pipeline Id
 - **plugin** - plugin type
 - **name** - plugin name (alias)
 - **status** - event status, `failed`, `rejected` or `accepted`

#### Summary `parser_plugin_processed_events`
Events statistic for parser.

Quantiles: 0.5, 0.9, 0.99, 1.0

Labels:
 - **pipeline** - pipeline Id
 - **plugin** - plugin type
 - **name** - plugin name (alias)
 - **status** - event status, `failed`, `rejected` or `accepted`

#### Summary `serializer_plugin_processed_events`
Events statistic for serializers.

Quantiles: 0.5, 0.9, 0.99, 1.0

Labels:
 - **pipeline** - pipeline Id
 - **plugin** - plugin type
 - **name** - plugin name (alias)
 - **status** - event status, `failed`, `rejected` or `accepted`

#### Summary `core_plugin_processed_events`
Events statistic for core plugins.

Quantiles: 0.5, 0.9, 0.99, 1.0

Labels:
 - **pipeline** - pipeline Id
 - **plugin** - plugin type
 - **name** - plugin name (alias)
 - **status** - event status, `failed`, `rejected` or `accepted`

## Plugins optional

Optional metrics that plugins may write. Usually, it's protocol-specific metrics.

#### Summary `plugin_http_server_requests_seconds`
Incoming http requests stats.

Quantiles: 0.5, 0.9, 0.99

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **uri** - incoming request path
 - **method** - incoming request method
 - **status** - request status

#### Summary `plugin_grpc_server_calls_seconds`
Handled RPCs stats.

Quantiles: 0.5, 0.9, 0.99

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **procedure** - method full name
 - **type** - RPC type
 - **status** - RPC status

#### Counter `plugin_grpc_server_called_total`
Total number of started RPCs.

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **procedure** - method full name
 - **type** - RPC type

#### Counter `plugin_grpc_server_completed_total`
Total number of completed RPCs.

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **procedure** - method full name
 - **type** - RPC type

#### Counter `plugin_grpc_server_received_messages_total`
Total number of received stream messages.

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **procedure** - method full name
 - **type** - RPC type

#### Counter `plugin_grpc_server_sent_messages_total`
Total number of sent stream messages.

Labels:
 - **pipeline** - pipeline Id
 - **plugin_name** - plugin name (alias)
 - **procedure** - method full name
 - **type** - RPC type
