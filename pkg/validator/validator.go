package validator

import (
	"regexp"

	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/non-standard/validators"
)

var v *validator.Validate = func() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterValidation("notblank", validators.NotBlank)
	return v
}()

// https://github.com/go-playground/validator/blob/master/regexes.go#L69
var splitParamsRegexp = regexp.MustCompile(`'[^']*'|\S+`)

// not used right now, but still may be usefull in feature
func Validate(s any) error {
	return v.Struct(s)
}

// func ManyOf(fl validator.FieldLevel) bool {
// 	if len(fl.Param()) == 0 {
// 		return false
// 	}

// 	vals := splitParamsRegexp.FindAllString(fl.Param(), -1)
// 	for i := 0; i < len(vals); i++ {
// 		vals[i] = strings.Replace(vals[i], "'", "", -1)
// 	}

// 	field := fl.Field()
// }
