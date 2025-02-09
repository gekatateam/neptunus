package template_text

import (
	"bytes"
	"fmt"
	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"log/slog"
	"text/template"
	"time"
)

type TemplateText struct {
	*core.BaseSerializer `mapstructure:"-"`
	TemplateText         string `mapstructure:"template_text"`
	Mode                 string `mapstructure:"mode"`
	DataOnly             bool   `mapstructure:"data_only"`
	OmitFailed           bool   `mapstructure:"omit_fail"`
	Header               string `mapstructure:"header"`
	Footer               string `mapstructure:"footer"`

	template *template.Template
	parser   func(events ...*core.Event) ([]byte, error)
}

func (t *TemplateText) Serialize(events ...*core.Event) ([]byte, error) {
	body, err := t.parser(events...)
	if err != nil {
		return nil, err
	}
	output := append([]byte(t.Header), body...)
	output = append(output, []byte(t.Footer)...)

	return output, nil
}

func (t *TemplateText) Close() error {
	return nil
}

func (t *TemplateText) Init() error {
	if t.TemplateText == "" {
		return fmt.Errorf("template text is empty")
	}

	tmp, err := template.New("text_template").Parse(t.TemplateText)
	if err != nil {
		return err
	}
	t.template = tmp

	switch t.Mode {
	case "batch":
		t.parser = t.parseBatch
	case "event":
		t.parser = t.parseByEvent
	default:
		return fmt.Errorf("forbidden mode: %v, expected one of: batch, event", t.Mode)
	}
	return nil
}

func init() {
	plugins.AddSerializer("template_text", func() core.Serializer {
		return &TemplateText{
			Mode:       "event",
			DataOnly:   false,
			OmitFailed: true,
		}
	})
}

func (t *TemplateText) parseBatch(events ...*core.Event) ([]byte, error) {
	now := time.Now()
	buff := bytes.NewBuffer(make([]byte, 0, 3072))

	err := t.template.Execute(buff, events)
	if err != nil {
		t.Observe(metrics.EventFailed, time.Since(now))
		t.Log.Error("batch template serialization failed", "error", err)
		return nil, err
	}
	t.Observe(metrics.EventAccepted, time.Since(now))
	return buff.Bytes(), nil
}

func (t *TemplateText) parseByEvent(events ...*core.Event) ([]byte, error) {
	now := time.Now()
	buff := bytes.NewBuffer(make([]byte, 0, 3072))
	var err error

	for _, e := range events {
		if t.DataOnly {
			err = t.template.Execute(buff, e.Data)
		} else {
			err = t.template.Execute(buff, e)
		}
		if err != nil {
			t.Observe(metrics.EventFailed, time.Since(now))
			t.Log.Error("serialization failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.AddTag("::template_serialization_failed")
			if t.OmitFailed {
				now = time.Now()
				continue
			}
			return nil, err
		}
		t.Observe(metrics.EventAccepted, time.Since(now))
		now = time.Now()
	}
	return buff.Bytes(), nil
}
