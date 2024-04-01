# Neptunus data model

The Neptunus engine works with events - single data frames. An event is a structure with seven main fields:
 - **Id** - unique ID of an event, which is usually generated when an event is created and may be replaced by data from request body, message, etc.
 - **Timestamp** - the time an event was created.
 - **Routing key** - an event key, which should be used for events routing inside a pipeline and in outer world; usually it is a queue/topic name, URL path, etc.
 - **Labels** - an event metadata map, which are used for routing with routing key; think of this as an event headers.
 - **Tags** - list of **unique** event attributes, which also can be used for routing; plugins adds special tags, e.g. `::starlark_processing_failed` when an error occurs.
- **Errors** - list of errors occurring in a pipeline; plugins add errors to an event if something goes wrong.
- **Fields** - an event payload, data map, that is filling by parsers; it is essentially the body of an event.

Also, each event has a **UUID** field that is randomly generated. This field is for internal use only and may be useful as an unique identifier.

## Event API

As a developer, you can use Event fields directly, however, in most cases it may be more convenient to use an [API](../core/event.go):
 - `SetLabel(key string, value string)` - add label to event; if label exist, it will be overwritten
 - `GetLabel(key string) (string, bool)` - get label value by key; if label does not exist, method returns false
 - `DeleteLabel(key string)` - delete label by key
 - `AddTag(tag string)` - add tag to event
 - `DeleteTag(tag string)` - delete tag from event
 - `HasTag(tag string) bool` - check if an event has the tag
 - `SetField(key string, value any) error` - set event field; if field cannot be set, error returns
 - `GetField(key string) (any, error)` - get event field; if field does not exist, error returns
 - `DeleteField(key string) error` - delete field from event; if field does not exist, error returns
 - `AppendFields(data Map)` - append fields to the root of data map
 - `Clone() *Event` - clone event
<!-- - `Copy() *Event` - copy event; new Id and Timestamp will be generated -->
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

You can set a tracker for each event using `SetHook(hook func())` method, but only once. Tracker creates with duty counter equal `1` at creation stage. That counter changes in two cases:
 - it increases when an event is cloned using corresponding method; cloned events shares tracker.
 - it decreases when an event `Done()` method calls.

When tracker duty counter decreases to zero, tracker will call hook function.

Tracker can be used by input plugins that wants to know when event processing done, such as `beats` or `kafka`, before responding to a client/broker that message has been accepted.

This also means than processors and outputs must call `Done()` method when an event no more needed, delivered or failed after configured attempts.
