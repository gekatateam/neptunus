package converter

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins/common/convert"
)

type from int

var fromFromString = map[string]from{
	"label":      fromLabel,
	"field":      fromField,
	"id":         fromId,
	"uuid":       fromUuid,
	"timestamp":  fromTimestamp,
	"routingkey": fromRoutingKey,
}

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
	toId to = iota + 1
	toRoutingKey
	toTimestamp
	toLabel
	toString
	toInteger
	toUnsigned
	toFloat
	toBoolean
	toTime
	toDuration
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
	case fromLabel:
		return c.fromLabel(e, p)

	case fromField:
		return c.fromField(e, p)

	case fromId:
		return c.fromId(e, p)

	case fromUuid:
		return c.fromUuid(e, p)

	case fromTimestamp:
		return c.fromTimestamp(e, p)

	case fromRoutingKey:
		return c.fromRoutingKey(e, p)

	default:
		panic(fmt.Errorf("unexpected from type: %v", p.from))
	}
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
		field, err := convert.AnyToStringWithTimeDuration(rawField, p.tlyt)
		if err != nil {
			return fmt.Errorf("from field: %v: %w", p.path, err)
		}
		e.Id = field
		return nil
	case toRoutingKey:
		field, err := convert.AnyToStringWithTimeDuration(rawField, p.tlyt)
		if err != nil {
			return fmt.Errorf("from field: %v: %w", p.path, err)
		}
		e.RoutingKey = field
		return nil
	case toTimestamp:
		field, err := convert.AnyToTime(rawField, p.tlyt)
		if err != nil {
			return fmt.Errorf("from field: %v: %w", p.path, err)
		}
		e.Timestamp = field
		return nil
	case toLabel:
		field, err := convert.AnyToStringWithTimeDuration(rawField, p.tlyt)
		if err != nil {
			return fmt.Errorf("from field: %v: %w", p.path, err)
		}
		e.SetLabel(p.path, field)
		return nil
	case toString:
		field, err = convert.AnyToStringWithTimeDuration(rawField, p.tlyt)
	case toInteger:
		field, err = convert.AnyToIntegerWithTimeDuration(rawField, p.tlyt)
	case toUnsigned:
		field, err = convert.AnyToUnsignedWithTimeDuration(rawField, p.tlyt)
	case toFloat:
		field, err = convert.AnyToFloat(rawField)
	case toBoolean:
		field, err = convert.AnyToBoolean(rawField)
	case toTime:
		field, err = convert.AnyToTime(rawField, p.tlyt)
	case toDuration:
		field, err = convert.AnyToDuration(rawField)
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

func (c *converter) fromUuid(e *core.Event, p conversionParams) error {
	switch p.to {
	case toId:
		e.Id = e.UUID.String()
		return nil
	case toRoutingKey:
		e.RoutingKey = e.UUID.String()
		return nil
	case toLabel:
		e.SetLabel(p.path, e.UUID.String())
		return nil
	case toString:
		if err := e.SetField(p.path, e.UUID.String()); err != nil {
			return fmt.Errorf("from uuid: set field failed: %w", err)
		}
		return nil
	}

	return fmt.Errorf("from uuid: cannot convert: unsopported target type: %v", p.to.String())
}

func (c *converter) fromLabel(e *core.Event, p conversionParams) error {
	label, ok := e.GetLabel(p.path)
	if !ok {
		return fmt.Errorf("from label: %v: no such label", p.path)
	}

	return c.fromStringType(e, p, label, "label")
}

func (c *converter) fromTimestamp(e *core.Event, p conversionParams) error {
	switch p.to {
	case toId:
		t, err := convert.AnyToStringWithTimeDuration(e.Timestamp, p.tlyt)
		if err != nil {
			return fmt.Errorf("from timestamp: %v: %w", p.path, err)
		}
		e.Id = t
		return nil
	case toRoutingKey:
		t, err := convert.AnyToStringWithTimeDuration(e.Timestamp, p.tlyt)
		if err != nil {
			return fmt.Errorf("from timestamp: %v: %w", p.path, err)
		}
		e.RoutingKey = t
		return nil
	case toTimestamp:
		return nil
	case toLabel:
		t, err := convert.AnyToStringWithTimeDuration(e.Timestamp, p.tlyt)
		if err != nil {
			return fmt.Errorf("from timestamp: %v: %w", p.path, err)
		}
		e.SetLabel(p.path, t)
		return nil
	case toString:
		t, err := convert.AnyToStringWithTimeDuration(e.Timestamp, p.tlyt)
		if err != nil {
			return fmt.Errorf("from timestamp: %v: %w", p.path, err)
		}

		if err := e.SetField(p.path, t); err != nil {
			return fmt.Errorf("from timestamp: set field failed: %w", err)
		}
		return nil
	case toInteger:
		t, err := convert.AnyToIntegerWithTimeDuration(e.Timestamp, p.tlyt)
		if err != nil {
			return fmt.Errorf("from timestamp: %v: %w", p.path, err)
		}

		if err := e.SetField(p.path, t); err != nil {
			return fmt.Errorf("from timestamp: set field failed: %w", err)
		}
		return nil
	case toUnsigned:
		t, err := convert.AnyToUnsignedWithTimeDuration(e.Timestamp, p.tlyt)
		if err != nil {
			return fmt.Errorf("from timestamp: %v: %w", p.path, err)
		}

		if err := e.SetField(p.path, t); err != nil {
			return fmt.Errorf("from timestamp: set field failed: %w", err)
		}
		return nil
	case toTime:
		if err := e.SetField(p.path, e.Timestamp); err != nil {
			return fmt.Errorf("from timestamp: set field failed: %w", err)
		}
		return nil
	}

	return fmt.Errorf("from timestamp: cannot convert to %v: unsopported type", p.to.String())
}

func (c *converter) fromId(e *core.Event, p conversionParams) error {
	return c.fromStringType(e, p, e.Id, "id")
}

func (c *converter) fromRoutingKey(e *core.Event, p conversionParams) error {
	return c.fromStringType(e, p, e.RoutingKey, "routing key")
}

func (c *converter) fromStringType(e *core.Event, p conversionParams, s, from string) error {
	var field any
	var err error

	switch p.to {
	case toId:
		e.Id = s
		return nil
	case toRoutingKey:
		e.RoutingKey = s
		return nil
	case toTimestamp:
		t, err := convert.AnyToTime(s, p.tlyt)
		if err != nil {
			return fmt.Errorf("from %v: %v: %w", from, p.path, err)
		}
		e.Timestamp = t
		return nil
	case toLabel:
		e.SetLabel(p.path, s)
		return nil
	case toString:
		field = s
	case toInteger:
		field, err = strconv.ParseInt(s, 0, 64)
	case toUnsigned:
		field, err = strconv.ParseUint(s, 0, 64)
	case toFloat:
		field, err = strconv.ParseFloat(s, 64)
	case toBoolean:
		field, err = strconv.ParseBool(s)
	case toDuration:
		field, err = time.ParseDuration(s)
	case toTime:
		field, err = convert.AnyToTime(s, p.tlyt)
	}

	if err == strconv.ErrRange && p.ioor {
		goto LABEL_OUT_OF_RANGE_IGNORED
	}

	if err != nil {
		return fmt.Errorf("from %v: %v: %w", from, p.path, err)
	}
LABEL_OUT_OF_RANGE_IGNORED:

	if err = e.SetField(p.path, field); err != nil {
		return fmt.Errorf("from %v: set field failed: %w", from, err)
	}

	return nil
}

func defaultLayout(layout string) string {
	if len(layout) == 0 {
		return time.RFC3339Nano
	}
	return layout
}
