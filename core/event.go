package core

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	Id         uuid.UUID         //`json:"id"`
	Timestamp  time.Time         //`json:"timestamp"`
	RoutingKey string            //`json:"routing_key"`
	Tags       []string          //`json:"tags"`
	Labels     map[string]string //`json:"labels"`
	Data       Map               //`json:"data"`
	Errors     []error
}

func NewEvent(routingkey string) *Event {
	return &Event{
		Id:         uuid.New(),
		Timestamp:  time.Now(),
		RoutingKey: routingkey,
		Tags:       make([]string, 5),
		Labels:     make(map[string]string),
		Data:       make(Map),
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
	}

	copy(event.Tags, e.Tags)
	for k, v := range e.Labels {
		event.Labels[k] = v
	}

	return &event
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
