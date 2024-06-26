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
