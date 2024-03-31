package converter

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
)

type conversionParams struct {
	from from
	to   to
	path string
}

type converter struct {}


