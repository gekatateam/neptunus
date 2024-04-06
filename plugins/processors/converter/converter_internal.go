package converter

import (
	"fmt"
	"strconv"

	"github.com/gekatateam/neptunus/core"
)

type from int

const (
	fromLabel from = iota + 1
	fromField
)

type to int

const (
	toTimestamp to = iota + 1

	toId
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
		var err   error

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
			if err != nil {
				return fmt.Errorf("from label to integer: %v: %w", p.path, err)
			}
		case toUnsigned:
			field, err = strconv.ParseUint(label, 0, 64)
			if err != nil {
				return fmt.Errorf("from label to unsigned: %v: %w", p.path, err)
			}
		case toFloat:
			field, err = strconv.ParseFloat(label, 64)
			if err != nil {
				return fmt.Errorf("from label to float: %v: %w", p.path, err)
			}
		case toBoolean:
			field, err = strconv.ParseBool(label)
			if err != nil {
				return fmt.Errorf("from label to boolean: %v: %w", p.path, err)
			}
		}

		if err = e.SetField(p.path, field); err != nil {
			return fmt.Errorf("from label, set field failed: %v: %w", p.path, err)
		}
	case fromField:
		rawField, getErr := e.GetField(p.path)
		if getErr != nil {
			return fmt.Errorf("from field: %v: %w", p.path, getErr)
		}

		var field any
		var err   error

		switch p.to {
		case toId:
			field, err := anyToString(rawField)
			if err != nil {
				return fmt.Errorf("from field to id: %v: %w", p.path, err)
			}
			e.Id = field
			return nil
		case toLabel:
			field, err := anyToString(rawField)
			if err != nil {
				return fmt.Errorf("from field to label: %v: %w", p.path, err)
			}
			e.SetLabel(p.path, field)
			return nil
		case toString:
			field, err = anyToString(rawField)
			if err != nil {
				return fmt.Errorf("from field to string: %v: %w", p.path, err)
			}
		case toInteger:
			field, err = anyToInteger(rawField)
			if err != nil {
				return fmt.Errorf("from field to integer: %v: %w", p.path, err)
			}
		case toUnsigned:
			field, err = anyToUnsigned(rawField)
			if err != nil {
				return fmt.Errorf("from field to unsigned: %v: %w", p.path, err)
			}
		case toFloat:
		case toBoolean:
		}

		if err = e.SetField(p.path, field); err != nil {
			return fmt.Errorf("from field, set field failed: %v: %w", p.path, err)
		}
	default:
		panic(fmt.Errorf("unexpected from type: %v", p.from))
	}

	return nil
}

func anyToString(v any) (string, error) {
	switch t := v.(type) {
	case string:
		return t, nil
	case int:
		return strconv.FormatInt(int64(t), 10), nil
	case int8:
		return strconv.FormatInt(int64(t), 10), nil
	case int16:
		return strconv.FormatInt(int64(t), 10), nil
	case int32:
		return strconv.FormatInt(int64(t), 10), nil
	case int64:
		return strconv.FormatInt(int64(t), 10), nil
	case uint:
		return strconv.FormatUint(uint64(t), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(t), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(t), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(t), 10), nil
	case uint64:
		return strconv.FormatUint(uint64(t), 10), nil
	case float32:
		return strconv.FormatFloat(float64(t), 'f', -1, 64), nil
	case float64:
		return strconv.FormatFloat(float64(t), 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(t), nil
	default:
		return "", fmt.Errorf("cannot convert to string: unsupported type")
	}
}

func anyToInteger(v any) (int64, error) {
	switch t := v.(type) {
	case string:
		return strconv.ParseInt(t, 0, 64)
	case int:
		return int64(t), nil
	case int8:
		return int64(t), nil
	case int16:
		return int64(t), nil
	case int32:
		return int64(t), nil
	case int64:
		return int64(t), nil
	case uint:
		return int64(t), nil
	case uint8:
		return int64(t), nil
	case uint16:
		return int64(t), nil
	case uint32:
		return int64(t), nil
	case uint64:
		return int64(t), nil
	case float32:
		return int64(t), nil
	case float64:
		return int64(t), nil
	case bool:
		if t {
			return int64(1), nil
		} else {
			return int64(0), nil
		}
	default:
		return 0, fmt.Errorf("cannot convert to integer: unsupported type")
	}
}

func anyToUnsigned(v any) (uint64, error) {
	switch t := v.(type) {
	case string:
		return strconv.ParseUint(t, 0, 64)
	case int:
		return uint64(t), nil
	case int8:
		return uint64(t), nil
	case int16:
		return uint64(t), nil
	case int32:
		return uint64(t), nil
	case int64:
		return uint64(t), nil
	case uint:
		return uint64(t), nil
	case uint8:
		return uint64(t), nil
	case uint16:
		return uint64(t), nil
	case uint32:
		return uint64(t), nil
	case uint64:
		return uint64(t), nil
	case float32:
		return uint64(t), nil
	case float64:
		return uint64(t), nil
	case bool:
		if t {
			return uint64(1), nil
		} else {
			return uint64(0), nil
		}
	default:
		return 0, fmt.Errorf("cannot convert to unsigned: unsupported type")
	}
}

func anyToFloat(v any) (float64, error) {
	switch t := v.(type) {
	case string:
		return strconv.ParseFloat(t, 64)
	case int:
		return float64(t), nil
	case int8:
		return float64(t), nil
	case int16:
		return float64(t), nil
	case int32:
		return float64(t), nil
	case int64:
		return float64(t), nil
	case uint:
		return float64(t), nil
	case uint8:
		return float64(t), nil
	case uint16:
		return float64(t), nil
	case uint32:
		return float64(t), nil
	case uint64:
		return float64(t), nil
	case float32:
		return float64(t), nil
	case float64:
		return float64(t), nil
	case bool:
		if t {
			return float64(1), nil
		} else {
			return float64(0), nil
		}
	default:
		return 0, fmt.Errorf("cannot convert to float: unsupported type")
	}
}

func anyToBoolean(v any) (bool, error) {
	switch t := v.(type) {
	case string:
		return strconv.ParseBool(t)
	case int:
		return t > 0, nil
	case int8:
		return t > 0, nil
	case int16:
		return t > 0, nil
	case int32:
		return t > 0, nil
	case int64:
		return t > 0, nil
	case uint:
		return t > 0, nil
	case uint8:
		return t > 0, nil
	case uint16:
		return t > 0, nil
	case uint32:
		return t > 0, nil
	case uint64:
		return t > 0, nil
	case float32:
		return t > 0, nil
	case float64:
		return t > 0, nil
	case bool:
		return bool(t), nil
	default:
		return false, fmt.Errorf("cannot convert to boolean: unsupported type")
	}
}
