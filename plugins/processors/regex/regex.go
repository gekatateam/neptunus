package regex

import (
	"fmt"
	"regexp"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
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
			return fmt.Errorf("label %v regex compilation failed: %v", k, err)
		}
		p.labels[k] = regex
	}

	for k, v := range p.Fields {
		regex, err := regexp.Compile(v)
		if err != nil {
			return fmt.Errorf("field %v regex compilation failed: %v", k, err)
		}
		p.fields[k] = regex
	}

	return nil
}

func (p *Regex) Self() any {
	return p
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
				// because a named capture cannot contain a dot in it's name
				// like "(?P<path.to.key>[a-z]+)" (it's a compilation error)
				// no error is possible here
				e.SetField(name, match[i])
			}
		}
	}
}

func init() {
	plugins.AddProcessor("regex", func() core.Processor {
		return &Regex{}
	})
}
