# Grpc Output Plugin

The `grpc` output plugin sends an events to external systems using gRPC. See [input.proto](../../common/grpc/input.proto) for more information. This plugin requires serializer.

Plugin can be configured for using one of three RPCs:
 - `one` - plugin passes each event in serializer and send it by unary call.
 - `bulk` - plugin sends a stream of data after every __interval__ or when __buffer__ of events is full.
 - `stream` - plugin sends an endless stream of events; when server sends **cancellation token** plugin closes stream, waits for a __sleep__ and reconnects.

## Configuration
