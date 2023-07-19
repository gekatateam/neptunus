package star

import (
	"go.starlark.net/starlark"
)

var eventMethods = map[string]builtinFunc{
	// routing key methods
	"getRK": getRoutingKey, // f() routingKey String
	"setRK": setRoutingKey, // f(routingKey String)

	// labels methods
	"setLabel": setLabel, // f(key, value String)
	"getLabel": getLabel, // f(key String) value String|None
	"delLabel": delLabel, // f(key String)

	// fields methods
	// "getField": getField, // f(path String) value Value|None
	// "setField": setField, // f(path String, value Value) Error|None
	// "delField": delField, //f(path String)

	// tags methods
	// "addTag": addTag, // f(tag String)
	// "delTag": delTag, // f(tag String)
	// "hasTag": hasTag, // f(tag String) Bool

	// object methods
	// "copy": copy, // f() Event
	// "clone": clone, // f() Event
}

type builtinFunc func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)

func getRoutingKey (_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(args) > 0 || len(kwargs) > 0 { // less checks goes faster
	// 	return starlark.None, fmt.Errorf("%v: method does not accept arguments", b.Name())
	// }

	e := b.Receiver().(*StarEvent)
	return starlark.String(e.E.RoutingKey), nil
}

func setRoutingKey (_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(kwargs) > 0 {
	// 	return starlark.None, fmt.Errorf("%v: method does not accept keyword arguments", b.Name())
	// }

	var rk string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &rk); err != nil {
		return starlark.None, err
	}

	b.Receiver().(*StarEvent).E.RoutingKey = rk
	return starlark.None, nil
}

func setLabel (_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(kwargs) > 0 {
	// 	return starlark.None, fmt.Errorf("%v: method does not accept keyword arguments", b.Name())
	// }

	var key, value string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 2, &key, &value); err != nil {
		return starlark.None, err
	}

	b.Receiver().(*StarEvent).E.AddLabel(key, value)
	return starlark.None, nil
}

func getLabel (_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(kwargs) > 0 {
	// 	return starlark.None, fmt.Errorf("%v: method does not accept keyword arguments", b.Name())
	// }

	var key string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &key); err != nil {
		return starlark.None, err
	}

	label, found := b.Receiver().(*StarEvent).E.GetLabel(key)
	if found {
		return starlark.String(label), nil
	}
	return starlark.None, nil
}

func delLabel (_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// if len(kwargs) > 0 {
	// 	return starlark.None, fmt.Errorf("%v: method does not accept keyword arguments", b.Name())
	// }

	var key string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &key); err != nil {
		return starlark.None, err
	}

	b.Receiver().(*StarEvent).E.DeleteLabel(key)
	return starlark.None, nil
}
