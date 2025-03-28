package cronjob

import (
	"fmt"
	"time"
	_ "time/tzdata"

	"github.com/robfig/cron/v3"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
)

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

func (i *Cronjob) Close() error {
	i.cron.Stop()
	return nil
}

func (i *Cronjob) Run() {
	for _, j := range i.Jobs {
		if j.Force {
			j.Run()
		}
	}

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
