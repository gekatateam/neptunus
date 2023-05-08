package model

import (
	"time"

	"github.com/google/uuid"

	stanymap "github.com/gekatateam/pipeline/core/navimap"
)

type Event struct {
	Id         uuid.UUID
	Timestamp  time.Time
	RoutingKey string
	Tags       []string
	Labels     map[string]string
	Data       stanymap.Map
}
