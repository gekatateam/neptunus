package convert

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

func AnyToString(v any) (string, error) {
	if t, ok := v.(fmt.Stringer); ok {
		return t.String(), nil
	}

	switch t := v.(type) {
	case []byte:
		return string(t), nil
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
		return "", fmt.Errorf("cannot convert to string: unsupported type: %T", v)
	}
}

func AnyToStringWithTimeDuration(v any, layout string) (string, error) {
	if t, ok := v.(time.Duration); ok {
		return t.String(), nil
	}

	if t, ok := v.(time.Time); ok {
		switch layout {
		case "unix":
			return strconv.FormatInt(t.Unix(), 10), nil
		case "unix_micro":
			return strconv.FormatInt(t.UnixMicro(), 10), nil
		case "unix_milli":
			return strconv.FormatInt(t.UnixMilli(), 10), nil
		default:
			return t.Format(layout), nil
		}
	}

	return AnyToString(v)
}

func AnyToInteger(v any) (int64, error) {
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
		if uint64(t) > math.MaxInt64 {
			return int64(t), strconv.ErrRange
		}
		return int64(t), nil
	case uint8:
		return int64(t), nil
	case uint16:
		return int64(t), nil
	case uint32:
		return int64(t), nil
	case uint64:
		if uint64(t) > math.MaxInt64 {
			return int64(t), strconv.ErrRange
		}
		return int64(t), nil
	case float32:
		if t < math.MinInt64 || t > math.MaxInt64 {
			return int64(t), strconv.ErrRange
		}
		return int64(t), nil
	case float64:
		if t < math.MinInt64 || t > math.MaxInt64 {
			return int64(t), strconv.ErrRange
		}
		return int64(t), nil
	case bool:
		if t {
			return int64(1), nil
		}
		return int64(0), nil
	default:
		return 0, fmt.Errorf("cannot convert to integer: unsupported type: %T", v)
	}
}

func AnyToIntegerWithTimeDuration(v any, layout string) (int64, error) {
	if t, ok := v.(time.Duration); ok {
		return int64(t), nil
	}

	if t, ok := v.(time.Time); ok {
		switch layout {
		case "unix":
			return t.Unix(), nil
		case "unix_micro":
			return t.UnixMicro(), nil
		case "unix_milli":
			return t.UnixMilli(), nil
		default:
			return 0, fmt.Errorf("cannot convert to integer: unsupported layout: %v", layout)
		}
	}

	return AnyToInteger(v)
}

func AnyToUnsigned(v any) (uint64, error) {
	switch t := v.(type) {
	case string:
		return strconv.ParseUint(t, 0, 64)
	case int:
		if t < 0 {
			return uint64(t), strconv.ErrRange
		}
		return uint64(t), nil
	case int8:
		if t < 0 {
			return uint64(t), strconv.ErrRange
		}
		return uint64(t), nil
	case int16:
		if t < 0 {
			return uint64(t), strconv.ErrRange
		}
		return uint64(t), nil
	case int32:
		if t < 0 {
			return uint64(t), strconv.ErrRange
		}
		return uint64(t), nil
	case int64:
		if t < 0 {
			return uint64(t), strconv.ErrRange
		}
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
		if t < 0 || t > math.MaxUint64 {
			return uint64(t), strconv.ErrRange
		}
		return uint64(t), nil
	case float64:
		if t < 0 || t > math.MaxUint64 {
			return uint64(t), strconv.ErrRange
		}
		return uint64(t), nil
	case bool:
		if t {
			return uint64(1), nil
		}
		return uint64(0), nil
	default:
		return 0, fmt.Errorf("cannot convert to unsigned: unsupported type: %T", v)
	}
}

func AnyToUnsignedWithTimeDuration(v any, layout string) (uint64, error) {
	if t, ok := v.(time.Duration); ok {
		return uint64(t), nil
	}

	if t, ok := v.(time.Time); ok {
		switch layout {
		case "unix":
			return uint64(t.Unix()), nil
		case "unix_micro":
			return uint64(t.UnixMicro()), nil
		case "unix_milli":
			return uint64(t.UnixMilli()), nil
		default:
			return 0, fmt.Errorf("cannot convert to integer: unsupported layout: %v", layout)
		}
	}

	return AnyToUnsigned(v)
}

func AnyToFloat(v any) (float64, error) {
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
		}
		return float64(0), nil
	default:
		return 0, fmt.Errorf("cannot convert to float: unsupported type: %T", v)
	}
}

func AnyToBoolean(v any) (bool, error) {
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
		return false, fmt.Errorf("cannot convert to boolean: unsupported type: %T", v)
	}
}

func AnyToTime(v any, layout string) (time.Time, error) {
	if t, ok := v.(time.Time); ok {
		return t, nil
	}

	switch layout {
	case "unix", "unix_micro", "unix_milli":
	default:
		if t, ok := v.(string); ok {
			return time.Parse(layout, t)
		}
		return time.Time{}, fmt.Errorf("cannot convert to Time with layout %v: unsopported type: %T", layout, v)
	}

	var mnsec int64
	switch t := v.(type) {
	case string:
		var err error
		mnsec, err = strconv.ParseInt(t, 0, 64)
		if err != nil {
			return time.Time{}, err
		}
	case int:
		mnsec = int64(t)
	case int8:
		mnsec = int64(t)
	case int16:
		mnsec = int64(t)
	case int32:
		mnsec = int64(t)
	case int64:
		mnsec = int64(t)
	case uint:
		mnsec = int64(t)
	case uint8:
		mnsec = int64(t)
	case uint16:
		mnsec = int64(t)
	case uint32:
		mnsec = int64(t)
	case uint64:
		mnsec = int64(t)
	default:
		return time.Time{}, fmt.Errorf("cannot convert to Time: unsopported type: %T", v)
	}

	switch layout {
	case "unix":
		return time.Unix(mnsec, 0), nil
	case "unix_micro":
		return time.UnixMicro(mnsec), nil
	case "unix_milli":
		return time.UnixMilli(mnsec), nil
	}

	// should be unreachable due to previous check
	return time.Time{}, fmt.Errorf("cannot convert to Time: unsopported type: %T", v)
}

func AnyToDuration(v any) (time.Duration, error) {
	if t, ok := v.(time.Duration); ok {
		return t, nil
	}

	var dur int64
	switch t := v.(type) {
	case string:
		return time.ParseDuration(t)
	case int:
		dur = int64(t)
	case int8:
		dur = int64(t)
	case int16:
		dur = int64(t)
	case int32:
		dur = int64(t)
	case int64:
		dur = int64(t)
	case uint:
		dur = int64(t)
	case uint8:
		dur = int64(t)
	case uint16:
		dur = int64(t)
	case uint32:
		dur = int64(t)
	case uint64:
		dur = int64(t)
	default:
		return 0, fmt.Errorf("cannot convert to Duration: unsopported type: %T", v)
	}

	return time.Duration(dur), nil
}
