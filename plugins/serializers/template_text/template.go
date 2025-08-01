package template_text

import (
	"bytes"
	"fmt"
	"os"
	"text/template"
	"time"

	sprig "github.com/go-task/slim-sprig/v3"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/elog"
	cte "github.com/gekatateam/neptunus/plugins/common/template"
)

type TemplateText struct {
	*core.BaseSerializer `mapstructure:"-"`
	TemplateText         string `mapstructure:"template_text"`
	TemplatePath         string `mapstructure:"template_path"`

	template *template.Template
}

func (t *TemplateText) Serialize(events ...*core.Event) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	return t.serializeBatch(buf, events...)
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

func (t *TemplateText) serializeBatch(buf *bytes.Buffer, events ...*core.Event) ([]byte, error) {
	now := time.Now()

	te := make([]cte.TEvent, 0, len(events))
	for _, e := range events {
		te = append(te, cte.New(e))
	}

	err := t.template.Execute(buf, te)
	for _, e := range events {
		if err != nil {
			t.Log.Error("template serialization failed",
				"error", err,
				elog.EventGroup(e),
			)
			t.Observe(metrics.EventFailed, time.Since(now))
		} else {
			t.Log.Debug("event processed",
				elog.EventGroup(e),
			)
			t.Observe(metrics.EventAccepted, time.Since(now))
		}
		now = time.Now()
	}
	return buf.Bytes(), err
}

func init() {
	plugins.AddSerializer("template_text", func() core.Serializer {
		return &TemplateText{}
	})
}
