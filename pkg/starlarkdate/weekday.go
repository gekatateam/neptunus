package starlarkdate

import (
	"fmt"
	"strings"
	"time"

	startime "go.starlark.net/lib/time"
	"go.starlark.net/starlark"
)

var weekdays = map[string]Weekday{
	"sunday":    Weekday(time.Sunday),
	"monday":    Weekday(time.Monday),
	"tuesday":   Weekday(time.Tuesday),
	"wednesday": Weekday(time.Wednesday),
	"thursday":  Weekday(time.Thursday),
	"friday":    Weekday(time.Friday),
	"saturday":  Weekday(time.Saturday),
}

type Weekday time.Weekday

func (w Weekday) String() string { return time.Weekday(w).String() }

func (w Weekday) Type() string { return "date.weekday" }

func (w Weekday) Freeze() {}

func (w Weekday) Truth() starlark.Bool { return w >= 0 && w <= 6 }

func (w Weekday) Hash() (uint32, error) {
	return uint32(w), nil
}

func (d Weekday) Cmp(v starlark.Value, depth int) (int, error) {
	if x, y := d, v.(Weekday); x < y {
		return -1, nil
	} else if x > y {
		return 1, nil
	}
	return 0, nil
}

func WeekdayOf(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var ts startime.Time
	if err := starlark.UnpackPositionalArgs("weekday_of", args, kwargs, 1, &ts); err != nil {
		return starlark.None, err
	}

	return Weekday(time.Time(ts).Weekday()), nil
}

func ParseWeekday(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var w string
	if err := starlark.UnpackPositionalArgs("parse_weekday", args, kwargs, 1, &w); err != nil {
		return starlark.None, err
	}

	weekday, ok := weekdays[strings.ToLower(w)]
	if !ok {
		return starlark.None, fmt.Errorf("unknown weekday: %v", w)
	}

	return weekday, nil
}
