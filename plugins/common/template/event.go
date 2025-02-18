package template

import (
	"time"

	"github.com/gekatateam/neptunus/core"
)

type TEvent struct {
	e *core.Event
}

func New(e *core.Event) TEvent {
	return TEvent{
		e: e,
	}
}

func (te TEvent) RoutingKey() string {
	return te.e.RoutingKey
}

func (te TEvent) Id() string {
	return te.e.Id
}

func (te TEvent) Timestamp() time.Time {
	return te.e.Timestamp
}

func (te TEvent) GetLabel(key string) string {
	val, _ := te.e.GetLabel(key)
	return val
}

func (te TEvent) GetField(key string) any {
	val, err := te.e.GetField(key)
	if err != nil {
		return nil
	}

	return val
}
