package cronjob

import (
	"log/slog"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
)

type Job struct {
	*core.BaseInput `mapstructure:"-"`
	Name            string `mapstructure:"name"`
	Schedule        string `mapstructure:"schedule"`
	Force           bool   `mapstructure:"force"`
}

func (j *Job) Run() {
	now := time.Now()

	e := core.NewEvent("cronjob." + j.Name)
	e.SetLabel("schedule", j.Schedule)
	j.Out <- e

	j.Log.Debug("event produced",
		slog.Group("event",
			"id", e.Id,
			"key", e.RoutingKey,
		),
	)
	j.Observe(metrics.EventAccepted, time.Since(now))
}
