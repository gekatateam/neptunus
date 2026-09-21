# Neptunus data model

The Neptunus engine works with events - individual data frames. An event is a structure with seven main fields:
 - **Id** - the unique ID of an event. It is usually generated when an event is created and may be replaced with data from a request body, message, etc.
 - **Timestamp** - the time when an event was created.
 - **Routing key** - an event key used for routing events inside a pipeline and in the outside world. It is usually a queue or topic name, URL path, etc.
 - **Labels** - a map of event metadata used for routing together with the routing key; think of it as event headers.
 - **Tags** - a list of **unique** event attributes that can also be used for routing.
 - **Errors** - a list of errors that occurred in a pipeline; plugins add errors to an event if something goes wrong.
 - **Data** - an event payload, map, or slice populated by parsers; it is essentially the body of an event.

Each event also has a randomly generated **UUID** field. This field is for internal use only, but may be useful as an unique identifier.

## Event API

As a developer, you can use Event fields directly. However, in most cases, it may be more convenient to use the [API](../core/event.go):
 - `SetLabel(key string, value string)` - add a label to an event; if the label exists, it is overwritten
 - `GetLabel(key string) (string, bool)` - get a label value by key; if the label does not exist, the method returns false
 - `DeleteLabel(key string)` - delete a label by key
 - `AddTag(tag string)` - add a tag to an event
 - `DeleteTag(tag string)` - delete a tag from an event
 - `HasTag(tag string) bool` - check whether an event has a tag
 - `SetField(key string, value any) error` - set an event field; if the field cannot be set, an error is returned
 - `GetField(key string) (any, error)` - get an event field; if the field does not exist, an error is returned
 - `DeleteField(key string) error` - delete a field from an event; if the field does not exist, an error is returned
 - `Clone() *Event` - clone an event
 - `Done()` - mark an event as delivered, deleted from the pipeline, or permanently failed
 - `Duty() int32` - get the event duty counter value
 - `StackError(err error)` - add an error to an event

`SetField`, `GetField` and `DeleteField` use dots as path separator. For example:
```json
# event data before
{
    "message": "user login",
    "metadata": {
        "user": {
            "name.full": "John Doe",
            "email": "johndoe@gmail.com",
            "roles": [ "employee", "manager" ]
        }
    }
}
```
To get first user role, call `GetField("metadata.user.roles.0")`, to add a new field with age, call `SetField("metadata.user.age", 42)`.
```json
# event data after
{
    "message": "user login",
    "metadata": {
        "user": {
            "name.full": "John Doe",
            "email": "johndoe@gmail.com",
            "roles": [ "employee", "manager" ],
            "age": 42
        }
    }
}
```

The path separator can be escaped with a backslash, and a backslash can also be escaped with another backslash: `GetField("metadata.user.name\\.full")`.

Negative indexes are also supported for existing slices. In this case, the target element is at `len(slice) - |index|`. For example, you can get the last element from the slice `[0, 2, 4, 6]` using the index `-1`, because its length is `4` and `4 - 1 = 3`.

The following types can be used as field values: strings, integers (signed and unsigned), booleans, floating-point numbers, time, duration, arrays, slices, and maps with string keys. Other types may cause errors during serialization.

There are a few corner cases:
 - if `GetField(".")` is called, the method returns the event data as is.
 - if `DeleteField(".")` is called, the event data is set to `nil`.
 - if `SetField(".", value)` is called:
   - if the event data is `nil` - the event data will be set from the `value` argument;
   - if the event data and `value` argument are `map[string]any` - the `value` map will be merged into the event data;
   - if the event data and `value` argument are `[]any` - the `value` slice will be appended to the event data;
   - otherwise, an error is returned.

## Delivery Control

You can add delivery hooks for each event using the `AddHook(hook func())` method. Each call adds a new hook to the tracker. The tracker is created with a duty counter of `1`. That counter changes in two cases:
 - it increases when an event is cloned using the corresponding method; cloned events share the tracker.
 - it decreases when an event's `Done()` method is called.

When the duty counter decreases to zero, the tracker calls all hook functions in the order in which they were added.

The tracker can be used by input plugins that need to know when event's processing is complete, such as `beats` or `kafka`, before responding to a client or broker that the message has been accepted.

Plugins must **never** call the event's `Done()` method themselves; the pipeline does this automatically. Instead, processors must send unnecessary events to the `Drop` channel, and outputs must send processed events to `Done`.

In tests, you can use the event's `Duty()` method to make sure that a plugin works correctly with the tracker.
