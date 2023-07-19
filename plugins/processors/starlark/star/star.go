package star

import (
	"errors"
	"fmt"
	"sort"
	"starevent/core"

	"go.starlark.net/starlark"
)

var _ starlark.HasAttrs = &StarEvent{}

type StarEvent struct {
	E *core.Event
}

func (e *StarEvent) String() string {
	return fmt.Sprint(e.E)
}

func (e *StarEvent) Type() string {
	return "event"
}

func (e *StarEvent) Freeze() {}

func (e *StarEvent) Truth() starlark.Bool {
	return e.E != nil
}

func (e *StarEvent) Hash() (uint32, error) {
	return 0, errors.New("event not hashable")
}

func (e *StarEvent) Attr(name string) (starlark.Value, error) {
	attr, ok := eventMethods[name]
	if !ok {
		return nil, fmt.Errorf("event has no method %v", name)
	}

	return starlark.NewBuiltin(name, attr).BindReceiver(e), nil
}

func (e *StarEvent) AttrNames() []string {
	names := make([]string, 0, len(eventMethods))
	for name := range eventMethods {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func NewEvent(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return &StarEvent{E: core.NewEvent("testKey")}, nil
}
