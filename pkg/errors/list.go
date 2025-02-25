package errors

import "strings"

// https://pkg.go.dev/go/scanner#ErrorList
type Errorlist []error

func (e Errorlist) Error() string {
	var errs []string
	for _, v := range e {
		errs = append(errs, v.Error())
	}
	return strings.Join(errs, "; ")
}
