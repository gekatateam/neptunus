package starlark

import (
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/pkg/starlarkdate"
	starlarktime "go.starlark.net/lib/time"
	"go.starlark.net/starlark"
)

func init() {
	maps.Copy(rwEventMethods, roEventMethods)
	maps.Copy(rwEventMethods, woEventMethods)
}

var rwEventMethods = map[string]*starlark.Builtin{}

var roEventMethods = map[string]*starlark.Builtin{
	"getId":        starlark.NewBuiltin("getId", getId),               // f() id String
	"getTimestamp": starlark.NewBuiltin("getTimestamp", getTimestamp), // f() timestamp Time
	"getRK":        starlark.NewBuiltin("getRK", getRoutingKey),       // f() routingKey String
	"getLabel":     starlark.NewBuiltin("getLabel", getLabel),         // f(key String) value String|None
	"getField":     starlark.NewBuiltin("getField", getField),         // f(path String) value Value|None
	"hasTag":       starlark.NewBuiltin("hasTag", hasTag),             // f(tag String) Bool
	"getErrors":    starlark.NewBuiltin("getErrors", getErrors),       // f() List[String]
	"getUuid":      starlark.NewBuiltin("getUuid", getUuid),           // f() String
}

var woEventMethods = map[string]*starlark.Builtin{
	"setId":        starlark.NewBuiltin("setId", setId),               // f(id String)
	"setTimestamp": starlark.NewBuiltin("setTimestamp", setTimestamp), // f(timestamp Time)
	"setRK":        starlark.NewBuiltin("setRK", setRoutingKey),       // f(routingKey String)
	"setLabel":     starlark.NewBuiltin("setLabel", setLabel),         // f(key, value String)
	"delLabel":     starlark.NewBuiltin("delLabel", delLabel),         // f(key String)
	"setField":     starlark.NewBuiltin("setField", setField),         // f(path String, value Value) Error|None
	"delField":     starlark.NewBuiltin("delField", delField),         //f(path String)
	"addTag":       starlark.NewBuiltin("addTag", addTag),             // f(tag String)
	"delTag":       starlark.NewBuiltin("delTag", delTag),             // f(tag String)
	"shareTracker": starlark.NewBuiltin("shareTracker", shareTracker), // f(event Event)
	"delErrors":    starlark.NewBuiltin("delErrors", delErrors),       // f(event Event)
}

func getId(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return starlark.String(b.Receiver().(*Event).event.Id), nil
}

func setId(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var id string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &id); err != nil {
		return starlark.None, err
	}

	b.Receiver().(*Event).event.Id = id
	return starlark.None, nil
}

func getRoutingKey(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return starlark.String(b.Receiver().(*Event).event.RoutingKey), nil
}

func setRoutingKey(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var rk string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &rk); err != nil {
		return starlark.None, err
	}

	b.Receiver().(*Event).event.RoutingKey = rk
	return starlark.None, nil
}

func getTimestamp(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return starlarktime.Time(b.Receiver().(*Event).event.Timestamp), nil
}

func setTimestamp(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var ts starlarktime.Time
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &ts); err != nil {
		return starlark.None, err
	}

	b.Receiver().(*Event).event.Timestamp = time.Time(ts)
	return starlark.None, nil
}

func setLabel(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key, value string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &key, &value); err != nil {
		return starlark.None, err
	}

	b.Receiver().(*Event).event.SetLabel(key, value)
	return starlark.None, nil
}

func getLabel(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &key); err != nil {
		return starlark.None, err
	}

	label, found := b.Receiver().(*Event).event.GetLabel(key)
	if found {
		return starlark.String(label), nil
	}
	return starlark.None, nil
}

func delLabel(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &key); err != nil {
		return starlark.None, err
	}

	b.Receiver().(*Event).event.DeleteLabel(key)
	return starlark.None, nil
}

func getField(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &key); err != nil {
		return starlark.None, err
	}

	value, err := b.Receiver().(*Event).event.GetField(key)
	if err != nil {
		return starlark.None, nil
	}

	return ToStarlarkValue(value)
}

func setField(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key string
	var value starlark.Value
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &key, &value); err != nil {
		return starlark.None, err
	}

	goValue, err := ToGoValue(value)
	if err != nil {
		return starlark.None, err
	}

	if err := b.Receiver().(*Event).event.SetField(key, goValue); err != nil {
		return Error(err.Error()), nil
	}

	return starlark.None, nil
}

func delField(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &key); err != nil {
		return starlark.None, err
	}

	b.Receiver().(*Event).event.DeleteField(key)
	return starlark.None, nil
}

func addTag(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var tag string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &tag); err != nil {
		return starlark.None, err
	}

	b.Receiver().(*Event).event.AddTag(tag)
	return starlark.None, nil
}

func delTag(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var tag string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &tag); err != nil {
		return starlark.None, err
	}

	b.Receiver().(*Event).event.DeleteTag(tag)
	return starlark.None, nil
}

func hasTag(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var tag string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &tag); err != nil {
		return starlark.None, err
	}

	return starlark.Bool(b.Receiver().(*Event).event.HasTag(tag)), nil
}

func getErrors(_ *starlark.Thread, b *starlark.Builtin, _ starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	errs := make([]starlark.Value, 0)
	for _, err := range b.Receiver().(*Event).event.Errors {
		errs = append(errs, starlark.String(err.Error()))
	}

	return starlark.NewList(errs), nil
}

func delErrors(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	b.Receiver().(*Event).event.Errors = make(core.Errors, 0)
	return starlark.None, nil
}

func getUuid(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return starlark.String(b.Receiver().(*Event).event.UUID.String()), nil
}

//lint:ignore U1000 reserved for better days
func cloneEvent(_ *starlark.Thread, b *starlark.Builtin, _ starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	return &Event{event: b.Receiver().(*Event).event.Clone()}, nil
}

func shareTracker(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var receiver *Event
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &receiver); err != nil {
		return starlark.None, err
	}

	core.ShareTracker(b.Receiver().(*Event).event, receiver.event)
	return starlark.None, nil
}

// event data types mapping
// string <-> starlark.String; starlark.String(T) <-> string(starlark.String)
// bool <-> starlark.Bool; starlark.Bool(T) <-> bool(starlark.Bool)
// int(8|16|32|64) <-> starlark.Int; MakeInt64(T) <-> Int64()
// uint(8|16|32|64) <-> starlark.Int; MakeUint64(T) <-> Uint64()
// float(32|64) <-> starlark.Float; starlark.Float(T) <-> float64(starlark.Float)
// T[], T[N] <-> *starlark.List
// map[string]T <-> *starlark.Dict
// time.Time <-> starlark.Time
// time.Duration <-> starlark.Duration

func ToStarlarkValue(goValue any) (starlark.Value, error) {
	if goValue == nil {
		return starlark.None, nil
	}

	switch v := goValue.(type) {
	case string:
		return starlark.String(v), nil
	case bool:
		return starlark.Bool(v), nil
	case int:
		return starlark.MakeInt64(int64(v)), nil
	case int8:
		return starlark.MakeInt64(int64(v)), nil
	case int16:
		return starlark.MakeInt64(int64(v)), nil
	case int32:
		return starlark.MakeInt64(int64(v)), nil
	case int64:
		return starlark.MakeInt64(v), nil
	case uint:
		return starlark.MakeUint64(uint64(v)), nil
	case uint8:
		return starlark.MakeUint64(uint64(v)), nil
	case uint16:
		return starlark.MakeUint64(uint64(v)), nil
	case uint32:
		return starlark.MakeUint64(uint64(v)), nil
	case uint64:
		return starlark.MakeUint64(v), nil
	case float32:
		return starlark.Float(v), nil
	case float64:
		return starlark.Float(v), nil
	case time.Duration:
		return starlarktime.Duration(v), nil
	case time.Time:
		return starlarktime.Time(v), nil
	case []any:
		list := make([]starlark.Value, 0, len(v))
		for _, elem := range v {
			starValue, err := ToStarlarkValue(elem)
			if err != nil {
				return starlark.None, err
			}
			list = append(list, starValue)
		}
		return starlark.NewList(list), nil
	case map[string]any:
		dict := starlark.NewDict(len(v))
		for k, elem := range v {
			starValue, err := ToStarlarkValue(elem)
			if err != nil {
				return starlark.None, err
			}
			if err = dict.SetKey(starlark.String(k), starValue); err != nil {
				return starlark.None, err
			}
		}
		return dict, nil
	}

	return starlark.None, fmt.Errorf("%T not representable in starlark", goValue)
}

func ToGoValue(starValue starlark.Value) (any, error) {
	switch v := starValue.(type) {
	case starlark.NoneType:
		return nil, nil
	case starlark.String:
		return string(v), nil
	case starlark.Bool:
		return bool(v), nil
	case starlark.Int: // int, uint, both here
		if value, ok := v.Int64(); ok {
			return value, nil
		}

		if value, ok := v.Uint64(); ok {
			return value, nil
		}

		return nil, errors.New("unknown starlark Int representation")
	case starlark.Float:
		return float64(v), nil
	case *starlark.List:
		iter := v.Iterate()
		defer iter.Done()
		slice := make([]any, 0, v.Len())
		var starValue starlark.Value
		for iter.Next(&starValue) {
			goValue, err := ToGoValue(starValue)
			if err != nil {
				return nil, err
			}
			slice = append(slice, goValue)
		}
		return slice, nil
	case *starlark.Dict:
		datamap := make(map[string]any, v.Len())
		for _, starKey := range v.Keys() {
			goKey, ok := starKey.(starlark.String)
			if !ok { // event datamap key must be a string
				return nil, fmt.Errorf("%v must be a string, got %v", starKey, starKey.Type())
			}

			// since the search is based on a known key,
			// it is expected that the value will always be found
			starValue, _, _ := v.Get(starKey)
			goValue, err := ToGoValue(starValue)
			if err != nil {
				return nil, err
			}

			datamap[string(goKey)] = goValue
		}
		return datamap, nil
	case starlarktime.Time:
		return time.Time(v), nil
	case starlarktime.Duration:
		return time.Duration(v), nil
	case starlarkdate.Month:
		return v.String(), nil
	case starlarkdate.Weekday:
		return v.String(), nil
	}

	return nil, fmt.Errorf("%v is not representable as event data value", starValue.Type())
}
