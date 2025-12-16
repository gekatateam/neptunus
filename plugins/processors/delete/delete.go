package delete

import (
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Delete struct {
	*core.BaseProcessor `mapstructure:"-"`
	Labels              []string `mapstructure:"labels"`
	Fields              []string `mapstructure:"fields"`
}

func (p *Delete) Init() error {
	return nil
}

func (p *Delete) Close() error {
	return nil
}

func (p *Delete) Run() {
	var now time.Time
	for e := range p.In {
		now = time.Now()

		for _, label := range p.Labels {
			e.DeleteLabel(label)
		}

		for _, field := range p.Fields {
			e.DeleteField(field)
		}

		p.Out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("delete", func() core.Processor {
		return &Delete{}
	})
}
