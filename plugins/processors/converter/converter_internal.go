package converter

import (
	"fmt"
	"strconv"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins/common/convert"
)

type from int

const (
	fromLabel from = iota + 1
	fromField
)

type to int

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
)

type conversionParams struct {
	from from
	to   to
	path string
	ioor bool
}

type converter struct{}

func (c *converter) Convert(e *core.Event, p conversionParams) error {
	switch p.from {
	case fromLabel:
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
		case toLabel:
			// doesn't make sence
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
	case fromField:
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
	default:
		panic(fmt.Errorf("unexpected from type: %v", p.from))
	}

	return nil
}
