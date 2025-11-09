package rename

import (
	"fmt"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/elog"
)

type Rename struct {
	*core.BaseProcessor `mapstructure:"-"`
	DeleteOrigin        bool              `mapstructure:"delete_origin"`
	Labels              map[string]string `mapstructure:"labels"`
	Fields              map[string]string `mapstructure:"fields"`
}

func (p *Rename) Init() error {
	return nil
}

func (p *Rename) Close() error {
	return nil
}

func (p *Rename) Run() {
	for e := range p.In {
		now := time.Now()

		for to, from := range p.Labels {
			if value, ok := e.GetLabel(from); ok {
				e.SetLabel(to, value)

				if p.DeleteOrigin {
					e.DeleteLabel(from)
				}
			}
		}

		var hasError bool
		for to, from := range p.Fields {
			value, err := e.GetField(from)
			if err != nil {
				continue
			}

			if err := e.SetField(to, value); err != nil {
				p.Log.Error(fmt.Sprintf("set field %v failed", to),
					"error", err,
					elog.EventGroup(e),
				)
				hasError = true
				continue
			}

			if p.DeleteOrigin {
				e.DeleteField(from)
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
	plugins.AddProcessor("rename", func() core.Processor {
		return &Rename{}
	})
}
