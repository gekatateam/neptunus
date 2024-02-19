package ider

import (
	"errors"
	"fmt"
	"regexp"
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

var idConfigPattern = regexp.MustCompile(`^([a-z]+):([\w-\.]+)$`)

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

	match := idConfigPattern.FindStringSubmatch(i.IdFrom)
	if len(match) != 3 {
		return errors.New("id_from: bad format, expected 'source:path'")
	}

	i.source = match[1]
	i.path = match[2]

	if _, ok := sources[i.source]; !ok {
		return fmt.Errorf("id_from: unknown type: %v; expected one of: label, field", i.source)
	}

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
			case []byte:
				e.Id = string(value)
			}
		}
	}
}
