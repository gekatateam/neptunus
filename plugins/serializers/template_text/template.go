package template_text

import (
	"bytes"
	"fmt"
	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/go-task/slim-sprig/v3"
	"log/slog"
	"os"
	"text/template"
	"time"
)

type TemplateText struct {
	*core.BaseSerializer `mapstructure:"-"`
	TemplateText         string `mapstructure:"template_text"`
	TemplatePath         string `mapstructure:"template_path"`

	template *template.Template
}

func (t *TemplateText) Serialize(events ...*core.Event) ([]byte, error) {
	return t.parseBatch(events...)
}

func (t *TemplateText) Close() error {
	return nil
}

func (t *TemplateText) Init() error {
	var templateOutput string
	if len(t.TemplatePath) > 0 {
		rawOutput, err := os.ReadFile(t.TemplatePath)
		if err != nil {
			return fmt.Errorf("file reading failed: %w", err)
		}
		templateOutput = string(rawOutput)
	} else if len(t.TemplateText) > 0 {
		templateOutput = t.TemplateText
	} else {
		return fmt.Errorf("no template file or text defined")
	}

	tmp, err := template.New("text_template").Funcs(sprig.FuncMap()).Parse(templateOutput)
	if err != nil {
		return err
	}
	t.template = tmp

	return nil
}

func (t *TemplateText) parseBatch(events ...*core.Event) ([]byte, error) {
	now := time.Now()
	buff := bytes.NewBuffer(make([]byte, 0, 3072))

	err := t.template.Execute(buff, events)
	for _, event := range events {
		if err != nil {
			t.Observe(metrics.EventFailed, time.Since(now))
			t.Log.Error("template serialization failed", "error", err,
				slog.Group("event",
					"id", event.Id,
					"key", event.RoutingKey,
				),
			)
		} else {
			t.Observe(metrics.EventAccepted, time.Since(now))
			t.Log.Debug("event processed",
				slog.Group("event",
					"id", event.Id,
					"key", event.RoutingKey,
				),
			)
		}
	}
	return buff.Bytes(), err
}

func init() {
	plugins.AddSerializer("template_text", func() core.Serializer {
		return &TemplateText{}
	})
}
