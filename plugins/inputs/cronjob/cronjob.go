package cronjob

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Job struct {
	*core.BaseInput `mapstructure:"-"`
	Name            string `mapstructure:"name"`
	Schedule        string `mapstructure:"schedule"`

	out chan<- *core.Event
}

func (j *Job) Run() {
	now := time.Now()

	e := core.NewEvent("cronjob." + j.Name)
	e.SetLabel("schedule", j.Schedule)
	j.out <- e

	j.Log.Debug("event produced",
		slog.Group("event",
			"id", e.Id,
			"key", e.RoutingKey,
		),
	)
	j.Observe(metrics.EventAccepted, time.Since(now))
}

type Cronjob struct {
	*core.BaseInput `mapstructure:"-"`
	Location        string `mapstructure:"location"`
	Jobs            []*Job `mapstructure:"jobs"`

	cron *cron.Cron
}

func (i *Cronjob) Init() error {
	loc, err := time.LoadLocation(i.Location)
	if err != nil {
		return err
	}

	i.cron = cron.New(
		cron.WithLocation(loc),
		cron.WithSeconds(),
		cron.WithLogger(cron.DiscardLogger),
	)

	for _, j := range i.Jobs {
		j.BaseInput = i.BaseInput
		if _, err := i.cron.AddJob(j.Schedule, j); err != nil {
			return fmt.Errorf("job %v scheduling failed: %w", j.Name, err)
		}
	}
	return nil
}

func (i *Cronjob) SetChannels(out chan<- *core.Event) {
	for _, j := range i.Jobs {
		j.out = out
	}
}

func (i *Cronjob) Close() error {
	i.cron.Stop()
	return nil
}

func (i *Cronjob) Run() {
	i.cron.Run()
}

func init() {
	plugins.AddInput("cronjob", func() core.Input {
		return &Cronjob{
			Location: "UTC",
			Jobs:     []*Job{},
		}
	})
}
