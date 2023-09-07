package regex

import (
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
)

type Regex struct {
	alias  string
	pipe   string
	Labels map[string]string `mapstructure:"labels"`
	Fields map[string]string `mapstructure:"fields"`

	in  <-chan *core.Event
	out chan<- *core.Event
	log *slog.Logger

	fields map[string]*regexp.Regexp
	labels map[string]*regexp.Regexp
}

func (p *Regex) Init(config map[string]any, alias, pipeline string, log *slog.Logger) error {
	if err := mapstructure.Decode(config, p); err != nil {
		return err
	}

	p.alias = alias
	p.pipe = pipeline
	p.log = log
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

func (p *Regex) Prepare(
	in <-chan *core.Event,
	out chan<- *core.Event,
) {
	p.in = in
	p.out = out
}

func (p *Regex) Close() error {
	return nil
}

func (p *Regex) Alias() string {
	return p.alias
}

func (p *Regex) Run() {
	for e := range p.in {
		now := time.Now()
		p.process(e)
		p.out <- e
		metrics.ObserveProcessorSummary("regex", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
	}
}

// saved for example
// func (p *Regex) Run() {
// 	for e := range p.in {
// 		now := time.Now()
// 		if err := p.match(e); err != nil {
// 			e.StackError(err)
// 			metrics.ObserveProcessorSummary("regex", p.alias, p.pipe, metrics.EventFailed, time.Since(now))
// 			goto SEND_OUT
// 		}
// 		metrics.ObserveProcessorSummary("regex", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
// 	SEND_OUT:
// 		p.out <- e
// 	}
// }

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
				e.AddLabel(name, match[i])
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
