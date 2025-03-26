package starlark

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"time"

	startime "go.starlark.net/lib/time"
	"go.starlark.net/starlark"
)

func init() {
	maps.Copy(rwEventMethods, roEventMethods)
	maps.Copy(rwEventMethods, woEventMethods)
}

var rwEventMethods = map[string]*starlark.Builtin{}

var roEventMethods = map[string]*starlark.Builtin{
	// id methods
	"getId": starlark.NewBuiltin("getId", getId), // f() id String

	// timestamp methods
	"getTimestamp": starlark.NewBuiltin("getTimestamp", getTimestamp), // f() timestamp Time

	// routing key methods
	"getRK": starlark.NewBuiltin("getRK", getRoutingKey), // f() routingKey String

	// labels methods
	"getLabel": starlark.NewBuiltin("getLabel", getLabel), // f(key String) value String|None

	// fields methods
	"getField": starlark.NewBuiltin("getField", getField), // f(path String) value Value|None

	// tags methods
	"hasTag": starlark.NewBuiltin("hasTag", hasTag), // f(tag String) Bool

	"getErrors": starlark.NewBuiltin("getErrors", getErrors), // f() List[String]

	"getUuid": starlark.NewBuiltin("getUuid", getUuid), // f() String
}

var woEventMethods = map[string]*starlark.Builtin{
	// id methods
	"setId": starlark.NewBuiltin("setId", setId), // f(id String)

	// timestamp methods
	"setTimestamp": starlark.NewBuiltin("setTimestamp", setTimestamp), // f(timestamp Time)

	// routing key methods
	"setRK": starlark.NewBuiltin("setRK", setRoutingKey), // f(routingKey String)

	// labels methods
	"setLabel": starlark.NewBuiltin("setLabel", setLabel), // f(key, value String)
	"delLabel": starlark.NewBuiltin("delLabel", delLabel), // f(key String)

	// fields methods
	"setField": starlark.NewBuiltin("setField", setField), // f(path String, value Value) Error|None
	"delField": starlark.NewBuiltin("delField", delField), //f(path String)

	// tags methods
	"addTag": starlark.NewBuiltin("addTag", addTag), // f(tag String)
	"delTag": starlark.NewBuiltin("delTag", delTag), // f(tag String)
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
	return startime.Time(b.Receiver().(*Event).event.Timestamp), nil
}

func setTimestamp(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var rawTs starlark.Value
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &rawTs); err != nil {
		return starlark.None, err
	}

	ts, ok := rawTs.(startime.Time)
	if !ok {
		return starlark.None, errors.New("method accepts Time only")
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

	return toStarlarkValue(value)
}

func setField(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key string
	var value starlark.Value
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &key, &value); err != nil {
		return starlark.None, err
	}

	goValue, err := toGoValue(value)
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

func getUuid(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return starlark.String(b.Receiver().(*Event).event.UUID.String()), nil
}

// lint:ignore U1000 reserved for better days
func cloneEvent(_ *starlark.Thread, b *starlark.Builtin, _ starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	return &Event{event: b.Receiver().(*Event).event.Clone()}, nil
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

func toStarlarkValue(goValue any) (starlark.Value, error) {
	if goValue == nil {
		return starlark.None, nil
	}

	v := reflect.ValueOf(goValue)
	switch v.Kind() {
	case reflect.String:
		return starlark.String(v.String()), nil
	case reflect.Bool:
		return starlark.Bool(v.Bool()), nil
	case reflect.Int64:
		if dur, ok := v.Interface().(time.Duration); ok {
			return startime.Duration(dur), nil
		}
		return starlark.MakeInt64(v.Int()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return starlark.MakeInt64(v.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return starlark.MakeUint64(v.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return starlark.Float(v.Float()), nil
	case reflect.Slice, reflect.Array:
		length := v.Len()
		list := make([]starlark.Value, 0, length)
		for i := 0; i < length; i++ {
			starValue, err := toStarlarkValue(v.Index(i).Interface())
			if err != nil {
				return starlark.None, err
			}
			list = append(list, starValue)
		}
		return starlark.NewList(list), nil
	case reflect.Map:
		dict := starlark.NewDict(v.Len())
		iter := v.MapRange()
		for iter.Next() {
			starKey, err := toStarlarkValue(iter.Key().Interface())
			if err != nil {
				return starlark.None, err
			}
			starValue, err := toStarlarkValue(iter.Value().Interface())
			if err != nil {
				return starlark.None, err
			}
			if err = dict.SetKey(starKey, starValue); err != nil {
				return starlark.None, err
			}
		}
		return dict, nil
	case reflect.Struct:
		if time, ok := v.Interface().(time.Time); ok {
			return startime.Time(time), nil
		}
	}

	return starlark.None, fmt.Errorf("%v not representable in starlark", v.Kind())
}

func toGoValue(starValue starlark.Value) (any, error) {
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
		var slice []any
		var starValue starlark.Value
		for iter.Next(&starValue) {
			goValue, err := toGoValue(starValue)
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
			goValue, err := toGoValue(starValue)
			if err != nil {
				return nil, err
			}

			datamap[goKey.String()] = goValue
		}
		return datamap, nil
	case startime.Time:
		return time.Time(v), nil
	case startime.Duration:
		return time.Duration(v), nil
	}

	return nil, fmt.Errorf("%v is not representable as event data value", starValue.Type())
}
