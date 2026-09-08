// Package validators wires go-playground/validator into Echo and translates its
// errors into the API's field-level error format.
package validators

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/rera/property-platform/backend/internal/dto"
)

type Validator struct {
	v *validator.Validate
}

func New() *Validator {
	v := validator.New()
	// Report the JSON name of a field so API clients see the name they sent.
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})
	_ = v.RegisterValidation("mobile", validateIndianMobile)
	return &Validator{v: v}
}

// Validate satisfies echo.Validator.
func (val *Validator) Validate(i any) error {
	return val.v.Struct(i)
}

// Errors converts a validator error into the API field-error list. It returns
// nil when err is not a validation error.
func Errors(err error) []dto.FieldError {
	var ve validator.ValidationErrors
	if !asValidationErrors(err, &ve) {
		return nil
	}
	out := make([]dto.FieldError, 0, len(ve))
	for _, fe := range ve {
		out = append(out, dto.FieldError{Field: fe.Field(), Message: message(fe)})
	}
	return out
}

func asValidationErrors(err error, target *validator.ValidationErrors) bool {
	if ve, ok := err.(validator.ValidationErrors); ok {
		*target = ve
		return true
	}
	return false
}

func message(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "required_without":
		return fmt.Sprintf("This field is required when %s is absent", fe.Param())
	case "email":
		return "Invalid email address"
	case "mobile":
		return "Invalid mobile number"
	case "min":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("Must be at least %s characters", fe.Param())
		}
		return fmt.Sprintf("Must be at least %s", fe.Param())
	case "max":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("Must be at most %s characters", fe.Param())
		}
		return fmt.Sprintf("Must be at most %s", fe.Param())
	case "len":
		return fmt.Sprintf("Must be exactly %s characters", fe.Param())
	case "oneof":
		return fmt.Sprintf("Must be one of: %s", strings.ReplaceAll(fe.Param(), " ", ", "))
	case "gte":
		return fmt.Sprintf("Must be %s or greater", fe.Param())
	case "lte":
		return fmt.Sprintf("Must be %s or less", fe.Param())
	case "url":
		return "Must be a valid URL"
	default:
		return "Invalid value"
	}
}

// validateIndianMobile accepts 10-digit Indian numbers, optionally prefixed with
// +91 or 0.
func validateIndianMobile(fl validator.FieldLevel) bool {
	s := strings.TrimSpace(fl.Field().String())
	s = strings.TrimPrefix(s, "+91")
	s = strings.TrimPrefix(s, "91")
	s = strings.TrimPrefix(s, "0")
	if len(s) != 10 {
		return false
	}
	if s[0] < '6' || s[0] > '9' {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// NormalizeMobile strips country-code and leading-zero prefixes so a number is
// stored in exactly one canonical 10-digit form.
func NormalizeMobile(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.TrimPrefix(s, "+91")
	if len(s) == 12 && strings.HasPrefix(s, "91") {
		s = s[2:]
	}
	if len(s) == 11 && strings.HasPrefix(s, "0") {
		s = s[1:]
	}
	return s
}
