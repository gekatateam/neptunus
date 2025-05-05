package starlark

import (
	"errors"

	"go.starlark.net/starlark"
)

var _ starlark.Value = Error("")

type Error string

func (e Error) String() string {
	return string(e)
}

func (e Error) Type() string {
	return "error"
}

func (e Error) Freeze() {}

func (e Error) Truth() starlark.Bool {
	return len(e) > 0
}

func (e Error) Hash() (uint32, error) {
	return 0, errors.New("error not hashable")
}

func newError(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var msg string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &msg); err != nil {
		return starlark.None, err
	}

	return Error(msg), nil
}

// handle accepts Callable and wrap occured err into Error
func handle(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var fn starlark.Callable
	if err := starlark.UnpackPositionalArgs("handle", args, kwargs, 1, &fn); err != nil {
		return nil, err
	}

	result, err := starlark.Call(thread, fn, nil, nil)
	if err != nil {
		return Error(err.Error()), nil
	}
	return result, nil
}
