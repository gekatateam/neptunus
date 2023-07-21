package starlark

import (
	"errors"

	"go.starlark.net/starlark"
)

var _ starlark.Value = _error("")

type _error string

func (e _error) String() string {
	return string(e)
}

func (e _error) Type() string {
	return "error"
}

func (e _error) Freeze() {}

func (e _error) Truth() starlark.Bool {
	return len(e) > 0
}

func (e _error) Hash() (uint32, error) {
	return 0, errors.New("error not hashable")
}

func newError(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var msg string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &msg); err != nil {
		return starlark.None, err
	}

	return _error(msg), nil
}
