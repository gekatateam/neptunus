package converter

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins/common/convert"
)

type from int

const (
	fromLabel from = iota + 1
	fromField
	fromId
	fromUuid
	fromTimestamp
	fromRoutingKey
)

type to int

func (t to) String() string { return toAsString[t] }

var toAsString = map[to]string{
	toTimestamp:  "timestamp",
	toId:         "id",
	toRoutingKey: "routing_key",
	toLabel:      "label",
	toString:     "string",
	toInteger:    "integer",
	toUnsigned:   "unsigned",
	toFloat:      "float",
	toBoolean:    "boolean",
	toTime:       "time",
	toDuration:   "duration",
}

const (
	//lint:ignore U1000 reserved
	toTimestamp to = iota + 1

	toId
	toLabel
	toString
	toInteger
	toUnsigned
	toFloat
	toBoolean

	//lint:ignore U1000 reserved
	toTime
	toDuration
	toRoutingKey
)

type conversionParams struct {
	from from
	to   to
	path string
	tlyt string
	ioor bool
}

type converter struct{}

func (c *converter) Convert(e *core.Event, p conversionParams) error {
	switch p.from {

	// from label to type
	case fromLabel:
		return c.fromLabel(e, p)

	// from field to type
	case fromField:
		return c.fromField(e, p)

	// from id to type
	case fromId:
		return c.fromId(e, p)

	case fromUuid:
		if err := e.SetField(p.path, e.UUID.String()); err != nil {
			return fmt.Errorf("from uuid: set field failed: %w", err)
		}
	case fromTimestamp:
		if err := e.SetField(p.path, e.Timestamp); err != nil {
			return fmt.Errorf("from timestamp: set field failed: %w", err)
		}
	case fromRoutingKey:

	default:
		panic(fmt.Errorf("unexpected from type: %v", p.from))
	}

	return nil
}

func (c *converter) fromLabel(e *core.Event, p conversionParams) error {
	label, ok := e.GetLabel(p.path)
	if !ok {
		return fmt.Errorf("from label: %v: no such label", p.path)
	}

	var field any
	var err error

	switch p.to {
	case toId:
		e.Id = label
		return nil
	case toRoutingKey:
		e.RoutingKey = label
		return nil
	case toLabel: // doesn't make sense
		return nil
	case toTimestamp:
		t, err := convert.AnyToTime(label, p.tlyt)
		if err != nil {
			return fmt.Errorf("from label: %v: %w", p.path, err)
		}
		e.Timestamp = t
		return nil
	case toString:
		field = label
	case toInteger:
		// the true base is implied by the string's prefix following the sign (if present): 2 for "0b", 8 for "0" or "0o", 16 for "0x", and 10 otherwise
		field, err = strconv.ParseInt(label, 0, 64)
	case toUnsigned:
		field, err = strconv.ParseUint(label, 0, 64)
	case toFloat:
		field, err = strconv.ParseFloat(label, 64)
	case toBoolean:
		field, err = strconv.ParseBool(label)
	case toDuration:
		field, err = time.ParseDuration(label)
	case toTime:
		field, err = convert.AnyToTime(label, p.tlyt)
	}

	if err == strconv.ErrRange && p.ioor {
		goto LABEL_OUT_OF_RANGE_IGNORED
	}

	if err != nil {
		return fmt.Errorf("from label: %v: %w", p.path, err)
	}
LABEL_OUT_OF_RANGE_IGNORED:

	if err = e.SetField(p.path, field); err != nil {
		return fmt.Errorf("from label: set field failed: %w", err)
	}

	return nil
}

func (c *converter) fromField(e *core.Event, p conversionParams) error {
	rawField, getErr := e.GetField(p.path)
	if getErr != nil {
		return fmt.Errorf("from field: %v: %w", p.path, getErr)
	}

	var field any
	var err error

	switch p.to {
	case toId:
		field, err := convert.AnyToString(rawField)
		if err != nil {
			return fmt.Errorf("from field: %v: %w", p.path, err)
		}
		e.Id = field
		return nil
	case toRoutingKey:
		field, err := convert.AnyToString(rawField)
		if err != nil {
			return fmt.Errorf("from field: %v: %w", p.path, err)
		}
		e.RoutingKey = field
		return nil
	case toLabel:
		field, err := convert.AnyToString(rawField)
		if err != nil {
			return fmt.Errorf("from field: %v: %w", p.path, err)
		}
		e.SetLabel(p.path, field)
		return nil
	case toString:
		field, err = convert.AnyToString(rawField)
	case toInteger:
		field, err = convert.AnyToInteger(rawField)
	case toUnsigned:
		field, err = convert.AnyToUnsigned(rawField)
	case toFloat:
		field, err = convert.AnyToFloat(rawField)
	case toBoolean:
		field, err = convert.AnyToBoolean(rawField)
	}

	if err == strconv.ErrRange && p.ioor {
		goto FIELD_OUT_OF_RANGE_IGNORED
	}

	if err != nil {
		return fmt.Errorf("from field: %v: %w", p.path, err)
	}
FIELD_OUT_OF_RANGE_IGNORED:

	if err = e.SetField(p.path, field); err != nil {
		return fmt.Errorf("from field: set field failed: %w", err)
	}

	return nil
}

func (c *converter) fromId(e *core.Event, p conversionParams) error {
	var field any
	var err error

	switch p.to {
	case toId:
		// doesn't make sense
		return nil
	case toLabel:
		e.SetLabel(p.path, e.Id)
		return nil
	case toTimestamp:
		t, err := time.Parse(p.tlyt, e.Id)
		if err != nil {
			return fmt.Errorf("from label: %v: %w", p.path, err)
		}

		e.Timestamp = t
		return nil
	case toString:
		field = e.Id
	case toInteger:
		// the true base is implied by the string's prefix following the sign (if present): 2 for "0b", 8 for "0" or "0o", 16 for "0x", and 10 otherwise
		field, err = strconv.ParseInt(e.Id, 0, 64)
	case toUnsigned:
		field, err = strconv.ParseUint(e.Id, 0, 64)
	case toFloat:
		field, err = strconv.ParseFloat(e.Id, 64)
	case toBoolean:
		field, err = strconv.ParseBool(e.Id)
	case toDuration:
		field, err = time.ParseDuration(e.Id)
	case toTime:
		field, err = time.Parse(p.tlyt, e.Id)
	}

	if err == strconv.ErrRange && p.ioor {
		goto LABEL_OUT_OF_RANGE_IGNORED
	}

	if err != nil {
		return fmt.Errorf("from id: %v: %w", p.path, err)
	}
LABEL_OUT_OF_RANGE_IGNORED:

	if err = e.SetField(p.path, field); err != nil {
		return fmt.Errorf("from id: set field failed: %w", err)
	}

	return nil
}

func defaultLayout(layout string) string {
	if len(layout) == 0 {
		return time.RFC3339Nano
	}
	return layout
}
