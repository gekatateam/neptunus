package starlark

import (
	"errors"
	"fmt"
	"reflect"

	"go.starlark.net/starlark"

	"github.com/gekatateam/neptunus/core"
)

var eventMethods = map[string]builtinFunc{
	// routing key methods
	"getRK": getRoutingKey, // f() routingKey String
	"setRK": setRoutingKey, // f(routingKey String)

	// labels methods
	"addLabel": addLabel, // f(key, value String)
	"getLabel": getLabel, // f(key String) value String|None
	"delLabel": delLabel, // f(key String)

	// fields methods
	"getField": getField, // f(path String) value Value|None
	"setField": setField, // f(path String, value Value) Error|None
	"delField": delField, //f(path String)

	// tags methods
	"addTag": addTag, // f(tag String)
	"delTag": delTag, // f(tag String)
	"hasTag": hasTag, // f(tag String) Bool

	// object methods
	"copy":  copyEvent,  // f() Event
	"clone": cloneEvent, // f() Event
}

type builtinFunc func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)

func getRoutingKey(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(args) > 0 || len(kwargs) > 0 { // less checks goes faster
	// 	return starlark.None, fmt.Errorf("%v: method does not accept arguments", b.Name())
	// }

	e := b.Receiver().(*event)
	return starlark.String(e.event.RoutingKey), nil
}

func setRoutingKey(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(kwargs) > 0 {
	// 	return starlark.None, fmt.Errorf("%v: method does not accept keyword arguments", b.Name())
	// }

	var rk string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &rk); err != nil {
		return starlark.None, err
	}

	b.Receiver().(*event).event.RoutingKey = rk
	return starlark.None, nil
}

func addLabel(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(kwargs) > 0 {
	// 	return starlark.None, fmt.Errorf("%v: method does not accept keyword arguments", b.Name())
	// }

	var key, value string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &key, &value); err != nil {
		return starlark.None, err
	}

	b.Receiver().(*event).event.AddLabel(key, value)
	return starlark.None, nil
}

func getLabel(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(kwargs) > 0 {
	// 	return starlark.None, fmt.Errorf("%v: method does not accept keyword arguments", b.Name())
	// }

	var key string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &key); err != nil {
		return starlark.None, err
	}

	label, found := b.Receiver().(*event).event.GetLabel(key)
	if found {
		return starlark.String(label), nil
	}
	return starlark.None, nil
}

func delLabel(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(kwargs) > 0 {
	// 	return starlark.None, fmt.Errorf("%v: method does not accept keyword arguments", b.Name())
	// }

	var key string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &key); err != nil {
		return starlark.None, err
	}

	b.Receiver().(*event).event.DeleteLabel(key)
	return starlark.None, nil
}

func getField(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(kwargs) > 0 {
	// 	return starlark.None, fmt.Errorf("%v: method does not accept keyword arguments", b.Name())
	// }

	var key string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &key); err != nil {
		return starlark.None, err
	}

	value, err := b.Receiver().(*event).event.GetField(key)
	if err != nil {
		return starlark.None, nil
	}

	return toStarlarkValue(value)
}

func setField(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(kwargs) > 0 {
	// 	return starlark.None, fmt.Errorf("%v: method does not accept keyword arguments", b.Name())
	// }

	var key string
	var value starlark.Value
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &key, &value); err != nil {
		return starlark.None, err
	}

	goValue, err := toGoValue(value)
	if err != nil {
		return starlark.None, err
	}

	if err := b.Receiver().(*event).event.SetField(key, goValue); err != nil {
		return Error(err.Error()), nil
	}

	return starlark.None, nil
}

func delField(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(kwargs) > 0 {
	// 	return starlark.None, fmt.Errorf("%v: method does not accept keyword arguments", b.Name())
	// }

	var key string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &key); err != nil {
		return starlark.None, err
	}

	b.Receiver().(*event).event.DeleteField(key)
	return starlark.None, nil
}

func addTag(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(kwargs) > 0 {
	// 	return starlark.None, fmt.Errorf("%v: method does not accept keyword arguments", b.Name())
	// }

	var tag string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &tag); err != nil {
		return starlark.None, err
	}

	b.Receiver().(*event).event.AddTag(tag)
	return starlark.None, nil
}

func delTag(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(kwargs) > 0 {
	// 	return starlark.None, fmt.Errorf("%v: method does not accept keyword arguments", b.Name())
	// }

	var tag string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &tag); err != nil {
		return starlark.None, err
	}

	b.Receiver().(*event).event.DeleteTag(tag)
	return starlark.None, nil
}

func hasTag(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(kwargs) > 0 {
	// 	return starlark.None, fmt.Errorf("%v: method does not accept keyword arguments", b.Name())
	// }

	var tag string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &tag); err != nil {
		return starlark.None, err
	}

	return starlark.Bool(b.Receiver().(*event).event.HasTag(tag)), nil
}

func copyEvent(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(args) > 0 || len(kwargs) > 0 { // less checks goes faster
	// 	return starlark.None, fmt.Errorf("%v: method does not accept arguments", b.Name())
	// }

	return &event{event: b.Receiver().(*event).event.Copy()}, nil
}

func cloneEvent(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(args) > 0 || len(kwargs) > 0 { // less checks goes faster
	// 	return starlark.None, fmt.Errorf("%v: method does not accept arguments", b.Name())
	// }

	return &event{event: b.Receiver().(*event).event.Clone()}, nil
}

// event data types mapping
// string <-> starlark.String; starlark.String(T) <-> string(starlark.String)
// bool <-> starlark.Bool; starlark.Bool(T) <-> bool(starlark.Bool)
// int(8|16|32|64) <-> starlark.Int; MakeInt64(T) <-> Int64()
// uint(8|16|32|64) <-> starlark.Int; MakeUint64(T) <-> Uint64()
// float(32|64) <-> starlark.Float; starlark.Float(T) <-> float64(starlark.Float)
// T[], T[N] <-> *starlark.List
// map[string]T <-> *starlark.Dict

func toStarlarkValue(goValue any) (starlark.Value, error) {
	v := reflect.ValueOf(goValue)
	switch v.Kind() {
	case reflect.String:
		return starlark.String(v.String()), nil
	case reflect.Bool:
		return starlark.Bool(v.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
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
	default:
		return starlark.None, fmt.Errorf("%v not representable in starlark", v.Kind())
	}
}

func toGoValue(starValue starlark.Value) (any, error) {
	switch v := starValue.(type) {
	case starlark.String:
		return string(v), nil
	case starlark.Bool:
		return bool(v), nil
	case starlark.Int: // int, uint both here
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
		slice := []any{}
		iter := v.Iterate()
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
		datamap := make(core.Map, v.Len())
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

			if err := datamap.SetValue(string(goKey), goValue); err != nil {
				return nil, err
			}
		}
		return datamap, nil
	default:
		return nil, fmt.Errorf("%v is not representable as event data value", starValue.Type())
	}
}