package starlark

import (
	"errors"
	"fmt"
	"sort"

	"go.starlark.net/starlark"

	"github.com/gekatateam/neptunus/core"
)

var (
	_           starlark.HasAttrs = (*Event)(nil)
	rwAttrNames []string          = []string{}
	roAttrNames []string          = []string{}
)

type Event struct {
	event     *core.Event
	writeable bool
}

func ROEvent(e *core.Event) *Event {
	return &Event{
		event:     e,
		writeable: false,
	}
}

func RWEvent(e *core.Event) *Event {
	return &Event{
		event:     e,
		writeable: true,
	}
}

func (e *Event) Event() *core.Event {
	return e.event
}

func (e *Event) String() string {
	return fmt.Sprint(e.event)
}

func (e *Event) Type() string {
	return "event"
}

func (e *Event) Freeze() {}

func (e *Event) Truth() starlark.Bool {
	return e.event != nil
}

func (e *Event) Hash() (uint32, error) {
	return 0, errors.New("event not hashable")
}

func (e *Event) Attr(name string) (starlark.Value, error) {
	attr, ok := e.attr()[name]
	if !ok {
		return nil, fmt.Errorf("event has no method %v", name)
	}

	return attr.BindReceiver(e), nil
}

func (e *Event) attr() map[string]*starlark.Builtin {
	if e.writeable {
		return rwEventMethods
	} else {
		return roEventMethods
	}
}

func (e *Event) AttrNames() []string {
	if e.writeable {
		return rwAttrNames
	} else {
		return roAttrNames
	}
}

func newEvent(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var rk string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &rk); err != nil {
		return starlark.None, err
	}

	return &Event{event: core.NewEvent(rk), writeable: true}, nil
}

func init() {
	for name := range roEventMethods {
		roAttrNames = append(roAttrNames, name)
	}
	sort.Strings(roAttrNames)

	for name := range rwEventMethods {
		rwAttrNames = append(rwAttrNames, name)
	}
	sort.Strings(rwAttrNames)
}
