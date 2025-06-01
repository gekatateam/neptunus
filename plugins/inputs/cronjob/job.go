package cronjob

import (
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/elog"
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
		elog.EventGroup(e),
	)
	j.Observe(metrics.EventAccepted, time.Since(now))
}
