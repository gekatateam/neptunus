package elog

import (
	"log/slog"

	"github.com/gekatateam/neptunus/core"
)

func EventGroup(e *core.Event) slog.Attr {
	return slog.Group("event",
		"id", e.Id,
		"key", e.RoutingKey,
		"uuid", e.UUID.String(),
	)
}
