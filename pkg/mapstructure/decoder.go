package mapstructure

import (
	"reflect"
	"time"

	"kythe.io/kythe/go/util/datasize"

	"github.com/mitchellh/mapstructure"
)

func Decode(input any, output any, hooks ...mapstructure.DecodeHookFunc) error {
	hooks = append(hooks,
		ToTimeHookFunc(),
		ToTimeDurationHookFunc(),
		ToByteSizeHookFunc(),
	)

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata:         nil,
		DecodeHook:       mapstructure.ComposeDecodeHookFunc(hooks...),
		Result:           output,
		WeaklyTypedInput: true,
	})
	if err != nil {
		return err
	}

	return decoder.Decode(input)
}

func ToTimeHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any) (any, error) {
		if t != reflect.TypeOf(time.Time{}) {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			return time.Parse(time.RFC3339, data.(string))
		case reflect.Float64:
			return time.Unix(0, int64(data.(float64))*int64(time.Millisecond)), nil
		case reflect.Int64:
			return time.Unix(0, data.(int64)*int64(time.Millisecond)), nil
		default:
			return data, nil
		}
	}
}

func ToTimeDurationHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any) (any, error) {
		if t != reflect.TypeOf(time.Duration(5)) {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			return time.ParseDuration(data.(string))
		case reflect.Int64:
			return time.Duration(data.(int64)), nil
		default:
			return data, nil
		}
	}
}

func ToByteSizeHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any) (any, error) {
		if t != reflect.TypeOf(datasize.Size(5)) {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			return datasize.Parse(data.(string))
		case reflect.Uint64:
			return datasize.Size(data.(uint64)), nil
		default:
			return data, nil
		}
	}
}
