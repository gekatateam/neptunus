package starlark

import (
	"errors"
	"fmt"
	"sort"

	"github.com/gekatateam/neptunus/core"
	"go.starlark.net/starlark"
)

var (
	_         starlark.HasAttrs = &_event{}
	attrNames []string          = []string{}
)

type _event struct {
	event *core.Event
}

func (e *_event) String() string {
	return fmt.Sprint(e.event)
}

func (e *_event) Type() string {
	return "event"
}

func (e *_event) Freeze() {}

func (e *_event) Truth() starlark.Bool {
	return e.event != nil
}

func (e *_event) Hash() (uint32, error) {
	return 0, errors.New("event not hashable")
}

func (e *_event) Attr(name string) (starlark.Value, error) {
	attr, ok := eventMethods[name]
	if !ok {
		return nil, fmt.Errorf("event has no method %v", name)
	}

	return starlark.NewBuiltin(name, attr).BindReceiver(e), nil
}

func (e *_event) AttrNames() []string {
	return attrNames
}

func newEvent(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var rk string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &rk); err != nil {
		return starlark.None, err
	}

	return &_event{event: core.NewEvent(rk)}, nil
}

func init() {
	for name := range eventMethods {
		attrNames = append(attrNames, name)
	}
	sort.Strings(attrNames)
}
