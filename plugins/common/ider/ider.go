package ider

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gekatateam/neptunus/core"
)

const (
	none  string = ""
	label string = "label"
	field string = "field"
)

var sources = map[string]struct{}{
	"label": {},
	"field": {},
}

// Ider is a helper to keep origin event Id.
type Ider struct {
	IdFrom string `mapstructure:"id_from"`

	source string
	path   string
}

func (i *Ider) Init() error {
	if len(i.IdFrom) == 0 {
		return nil
	}

	var (
		source []rune
		path   []rune
		next   bool
	)

	for _, r := range i.IdFrom {
		if r == ':' && !next {
			next = true
			continue
		}

		if next {
			path = append(path, r)
		} else {
			source = append(source, r)
		}
	}

	if len(source) == 0 || len(path) == 0 {
		return errors.New("id_from: bad format, expected 'type:path'")
	}

	if _, ok := sources[string(source)]; !ok {
		return fmt.Errorf("id_from: unknown type: %v; expected one of: label, field", string(source))
	}

	i.path = string(path)
	i.source = string(source)

	return nil
}

func (i *Ider) Apply(e *core.Event) {
	switch i.source {
	case none:
	case label:
		if label, ok := e.GetLabel(i.path); ok {
			e.Id = label
		}
	case field:
		if rawField, err := e.GetField(i.path); err == nil {
			switch value := rawField.(type) {
			case int64:
				e.Id = strconv.FormatInt(value, 10)
			case int32:
				e.Id = strconv.FormatInt(int64(value), 10)
			case int:
				e.Id = strconv.FormatInt(int64(value), 10)
			case uint64:
				e.Id = strconv.FormatUint(value, 10)
			case uint32:
				e.Id = strconv.FormatUint(uint64(value), 10)
			case uint:
				e.Id = strconv.FormatUint(uint64(value), 10)
			case float64:
				e.Id = strconv.FormatFloat(value, 'f', -1, 64)
			case float32:
				e.Id = strconv.FormatFloat(float64(value), 'f', -1, 64)
			case string:
				e.Id = value
			}
		}
	}
}
