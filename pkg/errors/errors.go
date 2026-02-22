package errors

import (
	"errors"
	"strings"
)

// https://pkg.go.dev/go/scanner#ErrorList
type Errorlist []error

func (e Errorlist) Error() string {
	var errs []string
	for _, v := range e {
		errs = append(errs, v.Error())
	}
	return strings.Join(errs, "; ")
}

// Just like errors.AsType, but for cases when you don't care about the error itself,
// but just want to know if it is of a certain type.
func AsType[E error](err error) bool {
	_, ok := errors.AsType[E](err)
	return ok
}
