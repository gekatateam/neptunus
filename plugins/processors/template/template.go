package template

import (
	"bytes"
	"log/slog"
	"text/template"
	"time"

	sprig "github.com/go-task/slim-sprig/v3"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	cte "github.com/gekatateam/neptunus/plugins/common/template"
)

type Template struct {
	*core.BaseProcessor `mapstructure:"-"`
	Id                  string            `mapstructure:"id"`
	RoutingKey          string            `mapstructure:"routing_key"`
	Labels              map[string]string `mapstructure:"labels"`
	Fields              map[string]string `mapstructure:"fields"`

	id         *template.Template
	routingKey *template.Template
	labels     map[string]*template.Template
	fields     map[string]*template.Template

	buf *bytes.Buffer
}

func (p *Template) Init() error {
	if len(p.Id) > 0 {
		id, err := template.New("id").Funcs(sprig.FuncMap()).Parse(p.Id)
		if err != nil {
			return err
		}
		p.id = id
	}

	if len(p.RoutingKey) > 0 {
		rk, err := template.New("routingKey").Funcs(sprig.FuncMap()).Parse(p.RoutingKey)
		if err != nil {
			return err
		}
		p.routingKey = rk
	}

	p.labels = make(map[string]*template.Template)
	p.fields = make(map[string]*template.Template)

	for l, t := range p.Labels {
		lt, err := template.New("labels::" + l).Funcs(sprig.FuncMap()).Parse(t)
		if err != nil {
			return err
		}
		p.labels[l] = lt
	}

	for f, t := range p.Fields {
		ft, err := template.New("fields::" + f).Funcs(sprig.FuncMap()).Parse(t)
		if err != nil {
			return err
		}
		p.fields[f] = ft
	}

	p.buf = bytes.NewBuffer(make([]byte, 0, 1024))

	return nil
}

func (p *Template) Close() error {
	return nil
}

func (p *Template) Run() {
	for e := range p.In {
		now := time.Now()
		hasError := false
		te := cte.New(e)

		if len(p.Id) > 0 {
			if err := p.id.Execute(p.buf, te); err != nil {
				p.Log.Error("template exec failed",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				e.StackError(err)
				e.AddTag("::template_processing_failed")
				hasError = true
			} else {
				e.Id = p.buf.String()
			}
			p.buf.Reset()
		}

		if len(p.RoutingKey) > 0 {
			if err := p.routingKey.Execute(p.buf, te); err != nil {
				p.Log.Error("template exec failed",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				e.StackError(err)
				e.AddTag("::template_processing_failed")
				hasError = true
			} else {
				e.RoutingKey = p.buf.String()
			}
			p.buf.Reset()
		}

		for label, lt := range p.labels {
			if err := lt.Execute(p.buf, te); err != nil {
				p.Log.Error("template exec failed",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				e.StackError(err)
				e.AddTag("::template_processing_failed")
				hasError = true
			} else {
				e.SetLabel(label, p.buf.String())
			}
			p.buf.Reset()
		}

		for field, ft := range p.fields {
			if err := ft.Execute(p.buf, te); err != nil {
				p.Log.Error("template exec failed",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				e.StackError(err)
				e.AddTag("::template_processing_failed")
				hasError = true
			} else {
				if err := e.SetField(field, p.buf.String()); err != nil {
					p.Log.Error("template executed successfully, but field set failed",
						"error", err,
						slog.Group("event",
							"id", e.Id,
							"key", e.RoutingKey,
							"field", field,
						),
					)
					e.StackError(err)
					e.AddTag("::template_processing_failed")
					hasError = true
				}
			}
			p.buf.Reset()
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
	plugins.AddProcessor("template", func() core.Processor {
		return &Template{}
	})
}
