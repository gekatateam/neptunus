package regex

import (
	"fmt"
	"regexp"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/elog"
)

type Regex struct {
	*core.BaseProcessor `mapstructure:"-"`
	Labels              map[string]string `mapstructure:"labels"`
	Fields              map[string]string `mapstructure:"fields"`

	fields map[string]*regexp.Regexp
	labels map[string]*regexp.Regexp
}

func (p *Regex) Init() error {
	p.labels = make(map[string]*regexp.Regexp)
	p.fields = make(map[string]*regexp.Regexp)

	for k, v := range p.Labels {
		regex, err := regexp.Compile(v)
		if err != nil {
			return fmt.Errorf("label %v regex compilation failed: %w", k, err)
		}
		p.labels[k] = regex
	}

	for k, v := range p.Fields {
		regex, err := regexp.Compile(v)
		if err != nil {
			return fmt.Errorf("field %v regex compilation failed: %w", k, err)
		}
		p.fields[k] = regex
	}

	return nil
}

func (p *Regex) Close() error {
	return nil
}

func (p *Regex) Run() {
	for e := range p.In {
		now := time.Now()
		p.process(e)
		p.Out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func (p *Regex) process(e *core.Event) {
	for name, rex := range p.labels {
		label, found := e.GetLabel(name)
		if !found {
			continue
		}

		match := rex.FindStringSubmatch(label)
		if len(match) == 0 {
			continue
		}

		for i, name := range rex.SubexpNames() {
			if i != 0 && name != "" {
				e.SetLabel(name, match[i])
			}
		}
	}

	for name, rex := range p.fields {
		rawField, err := e.GetField(name)
		if err != nil {
			continue
		}

		field, ok := rawField.(string)
		if !ok {
			continue
		}

		match := rex.FindStringSubmatch(field)
		if len(match) == 0 {
			continue
		}

		for i, name := range rex.SubexpNames() {
			if i != 0 && name != "" {
				if err := e.SetField(name, match[i]); err != nil {
					p.Log.Warn("set field failed",
						"error", err,
						elog.EventGroup(e),
					)
				}
			}
		}
	}
}

func init() {
	plugins.AddProcessor("regex", func() core.Processor {
		return &Regex{}
	})
}
