package defaults

import (
	"fmt"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
)

type Defaults struct {
	alias  string
	pipe   string
	Labels map[string]string `mapstructure:"labels"`
	Fields map[string]any    `mapstructure:"fields"`

	in  <-chan *core.Event
	out chan<- *core.Event
	log logger.Logger
}

func (p *Defaults) Init(config map[string]any, alias, pipeline string, log logger.Logger) error {
	if err := mapstructure.Decode(config, p); err != nil {
		return err
	}

	p.alias = alias
	p.pipe = pipeline
	p.log = log

	return nil
}

func (p *Defaults) Prepare(
	in <-chan *core.Event,
	out chan<- *core.Event,
) {
	p.in = in
	p.out = out
}

func (p *Defaults) Close() error {
	return nil
}

func (p *Defaults) Alias() string {
	return p.alias
}

func (p *Defaults) Run() {
	for e := range p.in {
		now := time.Now()
		hasError := false

		for k, v := range p.Labels {
			if _, ok := e.GetLabel(k); !ok {
				e.AddLabel(k, v)
			}
		}

		for k, v := range p.Fields {
			if _, err := e.GetField(k); err != nil {
				if err := e.SetField(k, v); err != nil {
					p.log.Error("error set field %v", k)
					e.StackError(fmt.Errorf("error set field %v", k))
					e.AddTag("::defaults_processing_failed")
					hasError = true
				}
			}
		}

		p.out <- e
		if hasError {
			metrics.ObserveProcessorSummary("defaults", p.alias, p.pipe, metrics.EventFailed, time.Since(now))
		} else {
			metrics.ObserveProcessorSummary("defaults", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
		}
	}
}

func init() {
	plugins.AddProcessor("defaults", func() core.Processor {
		return &Defaults{}
	})
}
