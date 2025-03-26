package core

import (
	"maps"
	"time"

	"github.com/google/uuid"

	"github.com/gekatateam/mappath"
)

type Event struct {
	Id         string
	UUID       uuid.UUID // for internal usage only
	Timestamp  time.Time
	RoutingKey string
	Tags       []string
	Labels     map[string]string
	Data       any
	Errors     Errors
	tracker    *tracker
}

func NewEvent(routingKey string) *Event {
	return &Event{
		Id:         uuid.New().String(),
		UUID:       uuid.New(),
		Timestamp:  time.Now(),
		RoutingKey: routingKey,
		Tags:       make([]string, 0, 5),
		Labels:     make(map[string]string),
		Data:       nil,
	}
}

func NewEventWithData(routingKey string, data any) *Event {
	return &Event{
		Id:         uuid.New().String(),
		UUID:       uuid.New(),
		Timestamp:  time.Now(),
		RoutingKey: routingKey,
		Tags:       make([]string, 0, 5),
		Labels:     make(map[string]string),
		Data:       data,
	}
}

func (e *Event) AddHook(hook hookFunc) {
	if e.tracker == nil {
		e.tracker = newTracker(hook)
		return
	}
	e.tracker.AddHook(hook)
}

func (e *Event) Done() {
	if e.tracker != nil {
		e.tracker.Decreace()
	}
}

func (e *Event) Duty() int32 {
	if e.tracker != nil {
		return e.tracker.duty
	}
	return -1
}

func (e *Event) GetField(key string) (any, error) {
	return mappath.Get(e.Data, key)
}

func (e *Event) SetField(key string, value any) error {
	p, err := mappath.Put(e.Data, key, value)
	if err != nil {
		return err
	}
	e.Data = p
	return nil
}

func (e *Event) DeleteField(key string) error {
	p, err := mappath.Delete(e.Data, key)
	if err != nil {
		return err
	}
	e.Data = p
	return nil
}

func (e *Event) StackError(err error) {
	e.Errors = append(e.Errors, err)
}

func (e *Event) Clone() *Event {
	event := &Event{
		Id:         e.Id,
		UUID:       uuid.New(),
		Timestamp:  e.Timestamp,
		RoutingKey: e.RoutingKey,
		Tags:       make([]string, len(e.Tags)),
		Errors:     make(Errors, len(e.Errors)),
		Labels:     make(map[string]string, len(e.Labels)),
		Data:       mappath.Clone(e.Data),
	}

	if e.tracker != nil {
		event.tracker = e.tracker.Copy()
	}

	copy(event.Tags, e.Tags)
	copy(event.Errors, e.Errors)
	maps.Copy(event.Labels, e.Labels)

	return event
}

func (e *Event) GetLabel(key string) (string, bool) {
	value, ok := e.Labels[key]
	return value, ok
}

func (e *Event) SetLabel(key, value string) {
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
