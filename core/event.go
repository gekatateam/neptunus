package core

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	Id         uuid.UUID
	Timestamp  time.Time
	RoutingKey string
	Tags       []string
	Labels     map[string]string
	Data       Map
	Errors     []error
	ctx        context.Context
}

func NewEvent(routingKey string) *Event {
	return &Event{
		Id:         uuid.New(),
		Timestamp:  time.Now(),
		RoutingKey: routingKey,
		Tags:       make([]string, 0, 5),
		Labels:     make(map[string]string),
		Data:       make(Map),
		ctx:        context.Background(),
	}
}

func NewEventWithData(routingKey string, data Map) *Event {
	return &Event{
		Id:         uuid.New(),
		Timestamp:  time.Now(),
		RoutingKey: routingKey,
		Tags:       make([]string, 0, 5),
		Labels:     make(map[string]string),
		Data:       data,
		ctx:        context.Background(),
	}
}

func (e *Event) GetField(key string) (any, error) {
	return e.Data.GetValue(key)
}

func (e *Event) SetField(key string, value any) error {
	return e.Data.SetValue(key, value)
}

func (e *Event) DeleteField(key string) (any, error) {
	return e.Data.DeleteValue(key)
}

func (e *Event) StackError(err error) {
	e.Errors = append(e.Errors, err)
}

func (e *Event) Copy() *Event {
	event := Event{
		Id:         uuid.New(),
		Timestamp:  e.Timestamp,
		RoutingKey: e.RoutingKey,
		Tags:       make([]string, len(e.Tags)),
		Labels:     make(map[string]string, len(e.Labels)),
		Data:       e.Data.Clone(),
		ctx:        context.Background(),
	}

	copy(event.Tags, e.Tags)
	for k, v := range e.Labels {
		event.Labels[k] = v
	}

	return &event
}

func (e *Event) Clone() *Event {
	event := Event{
		Id:         e.Id,
		Timestamp:  e.Timestamp,
		RoutingKey: e.RoutingKey,
		Tags:       make([]string, len(e.Tags)),
		Labels:     make(map[string]string, len(e.Labels)),
		Data:       e.Data.Clone(),
		ctx:        e.ctx,
	}

	copy(event.Tags, e.Tags)
	for k, v := range e.Labels {
		event.Labels[k] = v
	}

	return &event
}

func (e *Event) Context() context.Context {
	return e.ctx
}

func (e *Event) ReplaceContext(ctx context.Context) {
	e.ctx = ctx
}

func (e *Event) GetLabel(key string) (string, bool) {
	value, ok := e.Labels[key]
	return value, ok
}

func (e *Event) AddLabel(key, value string) {
	e.Labels[key] = value
}

func (e *Event) DeleteLabel(key string) {
	delete(e.Labels, key)
}

func (e *Event) HasTag(tag string) bool {
	for _, v := range e.Tags {
		if v == tag {
			return true
		}
	}
	return false
}

func (e *Event) AddTag(tag string) {
	for _, v := range e.Tags {
		if v == tag {
			return
		}
	}
	e.Tags = append(e.Tags, tag)
}

func (e *Event) DeleteTag(tag string) {
	var index = -1
	for i, v := range e.Tags {
		if v == tag {
			index = i
			break
		}
	}

	if index > -1 {
		e.Tags[index] = e.Tags[len(e.Tags)-1]
		e.Tags = e.Tags[:len(e.Tags)-1]
	}
}
