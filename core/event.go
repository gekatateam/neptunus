package core

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	Id         string
	UUID       uuid.UUID // for internal usage only
	Timestamp  time.Time
	RoutingKey string
	Tags       []string
	Labels     map[string]string
	Data       Payload
	Errors     Errors
	ctx        context.Context
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
		ctx:        context.Background(),
	}
}

func NewEventWithData(routingKey string, data Payload) *Event {
	return &Event{
		Id:         uuid.New().String(),
		UUID:       uuid.New(),
		Timestamp:  time.Now(),
		RoutingKey: routingKey,
		Tags:       make([]string, 0, 5),
		Labels:     make(map[string]string),
		Data:       data,
		ctx:        context.Background(),
	}
}

func (e *Event) SetHook(hook hookFunc) {
	if e.tracker == nil {
		e.tracker = newTracker(hook)
	}
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
	val, err := FindInPayload(e.Data, key)
	if err != nil {
		return nil, ErrNoSuchField
	}
	return val, nil
}

func (e *Event) SetField(key string, value any) error {
	p, err := PutInPayload(e.Data, key, value)
	if err != nil {
		return ErrInvalidPath
	}
	e.Data = p
	return nil
}

func (e *Event) DeleteField(key string) error {
	p, err := DeleteFromPayload(e.Data, key)
	if err != nil {
		return ErrNoSuchField
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
		Labels:     make(map[string]string, len(e.Labels)),
		Data:       ClonePayload(e.Data),
		ctx:        e.ctx,
	}

	if e.tracker != nil {
		event.tracker = e.tracker.Copy()
	}

	copy(event.Tags, e.Tags)
	for k, v := range e.Labels {
		event.Labels[k] = v
	}

	return event
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
