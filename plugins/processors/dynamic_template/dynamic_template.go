package dynamic_template

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	sprig "github.com/go-task/slim-sprig/v3"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/elog"
	cachestats "github.com/gekatateam/neptunus/plugins/common/metrics"
	cte "github.com/gekatateam/neptunus/plugins/common/template"
)

type DynamicTemplate struct {
	*core.BaseProcessor `mapstructure:"-"`
	EnableMetrics       bool          `mapstructure:"enable_metrics"`
	TemplateTTL         time.Duration `mapstructure:"template_ttl"`
	Labels              []string      `mapstructure:"labels"`
	Fields              []string      `mapstructure:"fields"`

	buf *bytes.Buffer
}

func (p *DynamicTemplate) Init() error {
	cache.Reg()
	p.buf = bytes.NewBuffer(make([]byte, 0, 1024))
	return nil
}

func (p *DynamicTemplate) Close() error {
	cache.Leave()
	return nil
}

func (p *DynamicTemplate) Run() {
	clearTicker := time.NewTicker(time.Minute)
	if p.TemplateTTL == 0 {
		clearTicker.Stop()
	}

	if p.EnableMetrics {
		cachestats.RegisterCache(p.Pipeline, p.Alias, "templates", &cache)
		defer cachestats.UnregisterCache(p.Pipeline, p.Alias, "templates")
	}

MAIN_LOOP:
	for {
		select {
		case e, ok := <-p.In:
			if !ok {
				clearTicker.Stop()
				break MAIN_LOOP
			}

			now := time.Now()
			if p.process(e) {
				p.Observe(metrics.EventAccepted, time.Since(now))
			} else {
				p.Observe(metrics.EventFailed, time.Since(now))
			}
			p.Out <- e
		case <-clearTicker.C:
			cache.DropOlderThan(p.TemplateTTL)
		}
	}
}

func (p *DynamicTemplate) process(e *core.Event) (allOk bool) {
	hasError := false
	te := cte.New(e)

	for _, l := range p.Labels {
		labelRaw, ok := e.GetLabel(l)
		if !ok {
			continue
		}

		result, ok := p.processTemplate(e, te, fmt.Sprintf("label:%v", l), labelRaw)
		if !ok {
			hasError = true
			continue
		}
		e.SetLabel(l, result)
	}

	for _, f := range p.Fields {
		fieldRaw, err := e.GetField(f)
		if err != nil {
			continue
		}

		switch field := fieldRaw.(type) {
		case string:
			result, ok := p.processTemplate(e, te, fmt.Sprintf("field:%v", f), field)
			if !ok {
				hasError = true
				continue
			}
			e.SetField(f, result)
		case map[string]any:
			for k, v := range field {
				data, ok := v.(string)
				if !ok {
					p.Log.Warn(fmt.Sprintf("%v.%v is not a string", f, k),
						elog.EventGroup(e),
					)
					continue
				}

				result, ok := p.processTemplate(e, te, fmt.Sprintf("field:%v.%v", f, k), data)
				if !ok {
					hasError = true
					continue
				}
				field[k] = result
			}

			e.SetField(f, field)
		case []any:
			for i, v := range field {
				data, ok := v.(string)
				if !ok {
					p.Log.Warn(fmt.Sprintf("%v.%v is not a string", f, i),
						elog.EventGroup(e),
					)
					continue
				}

				result, ok := p.processTemplate(e, te, fmt.Sprintf("field:%v.%v", f, i), data)
				if !ok {
					hasError = true
					continue
				}
				field[i] = result
			}

			e.SetField(f, field)
		default:
			p.Log.Warn(fmt.Sprintf("%v is not a string, slice or map", f),
				elog.EventGroup(e),
			)
			continue
		}
	}

	return !hasError
}

func (p *DynamicTemplate) processTemplate(e *core.Event, te cte.TEvent, source, content string) (result string, ok bool) {
	t, ok := cache.Get(content)
	if !ok { // if there is no template, try to create it and put in cache
		var err error
		t, err = template.New("template").Funcs(sprig.FuncMap()).Parse(content)
		if err != nil {
			p.Log.Error("template parsing failed",
				"error", fmt.Errorf("%v: %w", source, err),
				elog.EventGroup(e),
			)
			e.StackError(fmt.Errorf("%v: %w", source, err))
			return "", false
		}

		cache.Put(content, t)
	}

	if err := t.Execute(p.buf, te); err != nil {
		p.Log.Error("template exec failed",
			"error", fmt.Errorf("%v: %w", source, err),
			elog.EventGroup(e),
		)
		e.StackError(fmt.Errorf("%v: %w", source, err))
		return "", false
	}

	defer p.buf.Reset()
	return p.buf.String(), true
}

func init() {
	plugins.AddProcessor("dynamic_template", func() core.Processor {
		return &DynamicTemplate{
			TemplateTTL: time.Hour,
		}
	})
}
