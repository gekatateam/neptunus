package defaults

import (
	"fmt"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/elog"
)

type Defaults struct {
	*core.BaseProcessor `mapstructure:"-"`
	Labels              map[string]string `mapstructure:"labels"`
	Fields              map[string]any    `mapstructure:"fields"`
}

func (p *Defaults) Init() error {
	return nil
}

func (p *Defaults) Close() error {
	return nil
}

func (p *Defaults) Run() {
	var now time.Time
	for e := range p.In {
		now = time.Now()

		for k, v := range p.Labels {
			if _, ok := e.GetLabel(k); !ok {
				e.SetLabel(k, v)
			}
		}

		hasError := false
		for k, v := range p.Fields {
			if _, err := e.GetField(k); err != nil {
				if err := e.SetField(k, v); err != nil {
					p.Log.Error("error set field",
						"error", fmt.Errorf("%v: %w", k, err),
						elog.EventGroup(e),
					)
					e.StackError(fmt.Errorf("error set field: %w", err))
					hasError = true
				}
			}
		}

		p.Out <- e
		if hasError {
			p.Observe(metrics.EventFailed, time.Since(now))
		} else {
			p.Observe(metrics.EventAccepted, time.Since(now))
		}
	}
}

func init() {
	plugins.AddProcessor("defaults", func() core.Processor {
		return &Defaults{}
	})
}
