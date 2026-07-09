package mapstructure

import (
	"database/sql"
	"fmt"
	"reflect"
	"regexp"
	"time"

	"kythe.io/kythe/go/util/datasize"

	"github.com/go-viper/mapstructure/v2"
)

const unknownTypeErrorFormat = "cannot decode value of type %s into type %s"

func Decode(input any, output any, hooks ...mapstructure.DecodeHookFunc) error {
	hooks = append(hooks,
		ToTimeHookFunc(),
		ToTimeDurationHookFunc(),
		ToByteSizeHookFunc(),
		ToRuneHookFunc(),
		ToRegexpHookFunc(),
		ToSQLIsolationLevelHookFunc(),
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
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if t != reflect.TypeFor[time.Time]() {
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
			return nil, fmt.Errorf(unknownTypeErrorFormat, f, t)
		}
	}
}

func ToTimeDurationHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if t != reflect.TypeFor[time.Duration]() {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			return time.ParseDuration(data.(string))
		case reflect.Int64:
			return time.Duration(data.(int64)), nil
		default:
			return nil, fmt.Errorf(unknownTypeErrorFormat, f, t)
		}
	}
}

func ToByteSizeHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if t != reflect.TypeFor[datasize.Size]() {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			return datasize.Parse(data.(string))
		case reflect.Uint64:
			return datasize.Size(data.(uint64)), nil
		default:
			return nil, fmt.Errorf(unknownTypeErrorFormat, f, t)
		}
	}
}

func ToRuneHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if t != reflect.TypeFor[rune]() {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			r := []rune(data.(string))
			if len(r) != 1 {
				return nil, fmt.Errorf("cannot convert string of length %d to rune", len(r))
			}
			return r[0], nil
		case reflect.Int32:
			return rune(data.(int32)), nil
		default:
			return nil, fmt.Errorf(unknownTypeErrorFormat, f, t)
		}
	}
}

func ToRegexpHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if t != reflect.TypeFor[*regexp.Regexp]() {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			return regexp.Compile(data.(string))
		default:
			return nil, fmt.Errorf(unknownTypeErrorFormat, f, t)
		}
	}
}

var txIsolationLevels = map[string]sql.IsolationLevel{
	"Default":         sql.LevelDefault,
	"ReadUncommitted": sql.LevelReadUncommitted,
	"ReadCommitted":   sql.LevelReadCommitted,
	"WriteCommitted":  sql.LevelWriteCommitted,
	"RepeatableRead":  sql.LevelRepeatableRead,
	"Snapshot":        sql.LevelSnapshot,
	"Serializable":    sql.LevelSerializable,
	"Linearizable":    sql.LevelLinearizable,
}

func ToSQLIsolationLevelHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if t != reflect.TypeFor[sql.IsolationLevel]() {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			if level, ok := txIsolationLevels[data.(string)]; ok {
				return level, nil
			}
			return nil, fmt.Errorf("unknown tx isolation level: %s", data.(string))
		default:
			return nil, fmt.Errorf(unknownTypeErrorFormat, f, t)
		}
	}
}
