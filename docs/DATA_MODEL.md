# Neptunus data model

The Neptunus engine works with events - single data frames. An event is a structure with seven main fields:
 - **Id** - UUID of an event, which is usually generated when an event is created.
 - **Timestamp** - the time an event was created.
 - **Routing key** - an event key, which should be used for events routing inside a pipeline and in outer world; usually it is a queue/topic name, URL path, etc.
 - **Labels** - an event metadata map, which are used for routing with routing key; think of this as an event headers.
 - **Tags** - list of **unique** event attributes, which also can be used for routing; plugins adds special tags, e.g. `::starlark_processing_failed` when an error occurs.
- **Errors** - list of errors occurring in a pipeline; plugins add errors to an event if something goes wrong.
- **Fields** - an event payload, data map, that is filling by parsers; it is essentially the body of an event.

## Event API

As a developer, you can use Event fields directly, however, in most cases it may be more convenient to use an [API](../core/event.go):
 - `AddLabel(key string, value string)` - add label to event; if label exist, it will be overwritten
 - `GetLabel(key string) (string, bool)` - get label value by key; if label does not exist, method returns false
 - `DeleteLabel(key string)` - delete label by key
 - `AddTag(tag string)` - add tag to event
 - `DeleteTag(tag string)` - delete tag from event
 - `HasTag(tag string) bool` - check if an event has the tag
 - `SetField(key string, value any) error` - set event field; if field cannot be set, error returns
 - `GetField(key string) (any, error)` - get event field; if field does not exist, error returns
 - `DeleteField(key string) (any, error)` - delete field from event; if field does not exist, error returns
 - `AppendFields(data Map)` - append fields to the root of data map
 - `Clone() *Event` - clone event
 - `Copy() *Event` - copy event; new Id and Timestamp will be generated
 - `Done()` - mark event as delivered, deleted from pipeline or finally failed
 - `StackError(err error)` - add error to event

`SetField`, `GetField` and `DeleteField` use dots as path separator. For example:
```json
# event data before
{
    "message": "user login",
    "metadata": {
        "user": {
            "name": "John Doe",
            "email": "johndoe@gmail.com"
        }
    }
}
```
To get user name, call `GetField("metadata.user.name")`, to add a new field with age, call `SetField("metadata.user.age", 42)`.
```json
# event data after
{
    "message": "user login",
    "metadata": {
        "user": {
            "name": "John Doe",
            "email": "johndoe@gmail.com",
            "age": 42
        }
    }
}
```

These types can be used as field values: strings, integers (signed and unsigned), booleans, floating point numbers, arrays, slices and maps with string as a key. Any other types may cause errors at the serialization stage.

## Delivery Control

Each new event has a tracker with duty counter equal `1` at creation stage. That counter changes in two cases:
 - it increases when an event is copied or cloned using corresponding method; copied or cloned events shares tracker.
 - it decreases when an event `Done()` method calls.

You can set a hook for an event using `SetHook(hook hookFunc, payload any)` method. When tracker duty counter decreases to zero, tracker will call hook function with passed payload as an argument.

This can typically be used by input plugins that want to monitor event processing and delivery, such as `rabbitmq` or `kafka`, before telling a client/broker that the message has been accepted.

It also means than processors and outputs must call `Done()` method when an event no more needed, delivered or failed after configured attemts.
