package validator

import (
	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/non-standard/validators"
)

var v *validator.Validate = func() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterValidation("notblank", validators.NotBlank)
	return v
}()

func Validate(s any) error {
	return v.Struct(s)
}
