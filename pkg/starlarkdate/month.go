package starlarkdate

import (
	"fmt"
	"strings"
	"time"

	startime "go.starlark.net/lib/time"
	"go.starlark.net/starlark"
)

var months = map[string]Month{
	"january":   Month(time.January),
	"february":  Month(time.February),
	"march":     Month(time.March),
	"april":     Month(time.April),
	"may":       Month(time.May),
	"june":      Month(time.June),
	"july":      Month(time.July),
	"august":    Month(time.August),
	"september": Month(time.September),
	"october":   Month(time.October),
	"november":  Month(time.November),
	"december":  Month(time.December),
}

type Month time.Month

func (m Month) String() string { return time.Month(m).String() }

func (m Month) Type() string { return "date.month" }

func (m Month) Freeze() {}

func (m Month) Truth() starlark.Bool { return m >= 1 && m <= 12 }

func (m Month) Hash() (uint32, error) {
	return uint32(m) ^ uint32(int64(m)>>32), nil
}

func (d Month) Cmp(v starlark.Value, depth int) (int, error) {
	if x, y := d, v.(Month); x < y {
		return -1, nil
	} else if x > y {
		return 1, nil
	}
	return 0, nil
}

func MonthOf(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var ts startime.Time
	if err := starlark.UnpackPositionalArgs("month_of", args, kwargs, 1, &ts); err != nil {
		return starlark.None, err
	}

	return Month(time.Time(ts).Month()), nil
}

func ParseMonth(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var m string
	if err := starlark.UnpackPositionalArgs("parse_month", args, kwargs, 1, &m); err != nil {
		return starlark.None, err
	}

	month, ok := months[strings.ToLower(m)]
	if !ok {
		return starlark.None, fmt.Errorf("unknown month: %v", m)
	}

	return month, nil
}
