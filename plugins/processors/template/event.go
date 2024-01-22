package template

import (
	"time"

	"github.com/gekatateam/neptunus/core"
)

type templatedEvent struct {
	e *core.Event
}

func (te *templatedEvent) RoutingKey() string {
	return te.e.RoutingKey
}

func (te *templatedEvent) Id() string {
	return te.e.Id
}

func (te *templatedEvent) Timestamp() time.Time {
	return te.e.Timestamp
}

func (te *templatedEvent) GetLabel(key string) string {
	val, _ := te.e.GetLabel(key)
	return val
}

func (te *templatedEvent) GetField(key string) any {
	val, err := te.e.GetField(key)
	if err != nil {
		return nil
	}

	return val
}
