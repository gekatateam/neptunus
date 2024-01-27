package pass

import (
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Pass struct {
	*core.BaseFilter `mapstructure:"-"`
}

func (f *Pass) Init() error {
	return nil
}

func (f *Pass) Close() error {
	return nil
}

func (f *Pass) Run() {
	for e := range f.In {
		now := time.Now()
		f.Acc <- e
		f.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddFilter("pass", func() core.Filter {
		return &Pass{}
	})
}
