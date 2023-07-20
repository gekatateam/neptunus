package starlark

import (
	"errors"
	"fmt"
	"sort"

	"github.com/gekatateam/neptunus/core"
	"go.starlark.net/starlark"
)

var (
	_         starlark.HasAttrs = &event{}
	attrNames []string          = []string{}
)

type event struct {
	event *core.Event
}

func (e *event) String() string {
	return e.event.String()
}

func (e *event) Type() string {
	return "event"
}

func (e *event) Freeze() {}

func (e *event) Truth() starlark.Bool {
	return e.event != nil
}

func (e *event) Hash() (uint32, error) {
	return 0, errors.New("event not hashable")
}

func (e *event) Attr(name string) (starlark.Value, error) {
	attr, ok := eventMethods[name]
	if !ok {
		return nil, fmt.Errorf("event has no method %v", name)
	}

	return starlark.NewBuiltin(name, attr).BindReceiver(e), nil
}

func (e *event) AttrNames() []string {
	return attrNames
}

func NewEvent(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var rk string
	if err := starlark.UnpackPositionalArgs("newEvent", args, kwargs, 1, &rk); err != nil {
		return starlark.None, err
	}

	return &event{event: core.NewEvent(rk)}, nil
}

func init() {
	for name := range eventMethods {
		attrNames = append(attrNames, name)
	}
	sort.Strings(attrNames)
}
