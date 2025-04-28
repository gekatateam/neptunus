package starlarkdate

import (
	"time"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

var Module = &starlarkstruct.Module{
	Name: "date",
	Members: starlark.StringDict{
		// month block
		"month_of":    starlark.NewBuiltin("month_of", MonthOf),
		"parse_month": starlark.NewBuiltin("parse_month", ParseMonth),

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

		// weekday block
		"weekday_of":    starlark.NewBuiltin("weekday_of", WeekdayOf),
		"parse_weekday": starlark.NewBuiltin("parse_weekday", ParseWeekday),

		"sunday":    Weekday(time.Sunday),
		"monday":    Weekday(time.Monday),
		"tuesday":   Weekday(time.Tuesday),
		"wednesday": Weekday(time.Wednesday),
		"thursday":  Weekday(time.Thursday),
		"friday":    Weekday(time.Friday),
		"saturday":  Weekday(time.Saturday),
	},
}
