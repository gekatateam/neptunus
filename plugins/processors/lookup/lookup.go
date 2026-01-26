package lookup

import (
	"fmt"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/convert"
	"github.com/gekatateam/neptunus/plugins/common/elog"
)

type Lookup struct {
	*core.BaseProcessor `mapstructure:"-"`
	Labels              map[string]string `mapstructure:"labels"`
	Fields              map[string]string `mapstructure:"fields"`

	l core.Lookup
}

func (p *Lookup) Init() error {
	return nil
}

func (p *Lookup) Close() error {
	return nil
}

func (p *Lookup) SetLookup(l core.Lookup) {
	p.l = l
}

func (p *Lookup) Run() {
	for e := range p.In {
		now := time.Now()
		hasError := false

		for k, v := range p.Labels {
			val, err := p.l.Get(v)
			if err != nil {
				p.Log.Error(fmt.Sprintf("label %v: failed to get lookup data by key %v", k, v),
					"error", err,
					elog.EventGroup(e),
				)
				e.StackError(fmt.Errorf("label %v: failed to get lookup data by key %v: %w", k, v, err))
				hasError = true
				continue
			}

			s, err := convert.AnyToString(val)
			if err != nil {
				p.Log.Error(fmt.Sprintf("label %v: key %v: failed to convert to string", k, v),
					"error", err,
					elog.EventGroup(e),
				)
				e.StackError(fmt.Errorf("label %v: key %v: failed to convert to string: %w", k, v, err))
				hasError = true
				continue
			}

			e.SetLabel(k, s)
		}

		for k, v := range p.Fields {
			val, err := p.l.Get(v)
			if err != nil {
				p.Log.Error(fmt.Sprintf("field %v: failed to get lookup data by key %v", k, v),
					"error", err,
					elog.EventGroup(e),
				)
				e.StackError(fmt.Errorf("field %v: failed to get lookup data by key %v: %w", k, v, err))
				hasError = true
				continue
			}

			if err := e.SetField(k, val); err != nil {
				p.Log.Error(fmt.Sprintf("field %v: key %v: failed to set field", k, v),
					"error", err,
					elog.EventGroup(e),
				)
				e.StackError(fmt.Errorf("field %v: key %v: failed to set field: %w", k, v, err))
				hasError = true
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
	plugins.AddProcessor("lookup", func() core.Processor {
		return &Lookup{}
	})
}
