package ider

import "github.com/gekatateam/neptunus/core"

type Ider struct {
	Label string `mapstructure:"id_label"`
	Field string `mapstructure:"id_field"`
}

func (i *Ider) Apply(e *core.Event) {
	if len(i.Field) > 0 {
		if rawField, err := e.GetField(i.Field); err == nil {
			if field, ok := rawField.(string); ok {
				e.Id = field
				return
			}
		}
	}

	if len(i.Label) > 0 {
		if label, ok := e.GetLabel(i.Label); ok {
			e.Id = label
			return
		}
	}
}
