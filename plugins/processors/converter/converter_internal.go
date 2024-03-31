package converter

import "github.com/gekatateam/neptunus/core"

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



	return nil
}
