package convert

import (
	"fmt"
	"math"
	"strconv"
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
		return "", fmt.Errorf("cannot convert to string: unsupported type")
	}
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
		return 0, fmt.Errorf("cannot convert to integer: unsupported type")
	}
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
		return 0, fmt.Errorf("cannot convert to unsigned: unsupported type")
	}
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
		return 0, fmt.Errorf("cannot convert to float: unsupported type")
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
		return false, fmt.Errorf("cannot convert to boolean: unsupported type")
	}
}
