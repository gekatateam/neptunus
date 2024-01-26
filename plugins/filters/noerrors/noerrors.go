package noerrors

import (
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type NoErrors struct {
	*core.BaseFilter `mapstructure:"-"`
}

func (f *NoErrors) Init() error {
	return nil
}

func (f *NoErrors) Self() any {
	return f
}

func (f *NoErrors) Close() error {
	return nil
}

func (f *NoErrors) Run() {
	for e := range f.In {
		now := time.Now()
		if len(e.Errors) > 0 {
			f.Rej <- e
			f.Observe(metrics.EventRejected, time.Since(now))
		} else {
			f.Acc <- e
			f.Observe(metrics.EventAccepted, time.Since(now))
		}
	}
}

func init() {
	plugins.AddFilter("noerrors", func() core.Filter {
		return &NoErrors{}
	})
}
