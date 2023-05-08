package model

import (
	"time"

	"github.com/google/uuid"

	"github.com/gekatateam/pipeline/core/navigator"
)

type Event struct {
	Id         uuid.UUID
	Timestamp  time.Time
	RoutingKey string
	Tags       []string
	Labels     map[string]string
	Data       navigator.Map
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

func (e *Event) AddLabel(key, value string) {
	e.Labels[key] = value
}

func (e *Event) DeleteLabel(key string) {
	delete(e.Labels, key)
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
