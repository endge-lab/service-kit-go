package validator

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	govalidator "github.com/go-playground/validator/v10"
)

type Validator interface {
	Validate(i any) error
	ValidateField(val any, tag string) error
}

type FieldErrors map[string]string

type ValidationError struct {
	Fields FieldErrors
}

func (e *ValidationError) Error() string {
	return "validation error"
}

func AsValidationErr(err error) (*ValidationError, bool) {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve, true
	}
	return nil, false
}

var (
	upperRegex = regexp.MustCompile(`[A-Z]`)
	lowerRegex = regexp.MustCompile(`[a-z]`)
	digitRegex = regexp.MustCompile(`[0-9]`)
)

func isValidPassword(s string) bool {
	return len(s) >= 8 &&
		upperRegex.MatchString(s) &&
		lowerRegex.MatchString(s) &&
		digitRegex.MatchString(s)
}

type CustomValidator struct {
	v *govalidator.Validate
}

func NewValidator() *CustomValidator {
	v := govalidator.New()

	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return ""
		}
		return name
	})
	_ = v.RegisterValidation("password", func(fl govalidator.FieldLevel) bool {
		s, ok := fl.Field().Interface().(string)
		if !ok {
			return false
		}
		return isValidPassword(s)
	})
	return &CustomValidator{v: v}
}

func (c *CustomValidator) Validate(i any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = fmt.Errorf("validator panic: %w", e)
			} else {
				err = fmt.Errorf("validator panic: %v", r)
			}
		}
	}()

	if err := c.v.Struct(i); err != nil {
		var verrs govalidator.ValidationErrors
		if !errors.As(err, &verrs) {
			return fmt.Errorf("unexpected validation err: %w", err)
		}
		fields := make(FieldErrors, len(verrs))
		for _, fe := range verrs {
			fields[fe.Field()] = c.getErrorMessage(fe)
		}
		return &ValidationError{Fields: fields}
	}
	return nil
}

func (c *CustomValidator) ValidateField(val any, tag string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = fmt.Errorf("validator panic: %w", e)
			} else {
				err = fmt.Errorf("validator panic: %v", r)
			}
		}
	}()

	if err := c.v.Var(val, tag); err != nil {
		var verr govalidator.ValidationErrors
		if !errors.As(err, &verr) {
			return fmt.Errorf("unexpected validation err: %w", err)
		}
		fields := make(FieldErrors, len(verr))
		for _, fe := range verr {
			key := fe.Field()
			if key == "" {
				key = "value"
			}
			fields[key] = c.getErrorMessage(fe)
		}
		return &ValidationError{Fields: fields}
	}
	return nil
}

func (c *CustomValidator) getErrorMessage(err govalidator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "required field"
	case "email":
		return "invalid email format"
	case "min":
		return "min length " + err.Param()
	case "max":
		return "max length " + err.Param()
	case "len":
		return "must be exactly " + err.Param() + " characters long"
	case "oneof":
		return "must be one of [" + err.Param() + "]"
	case "gte":
		return "must be greater than or equal to " + err.Param()
	case "gt":
		return "must be greater than " + err.Param()
	case "lte":
		return "must be less than or equal to " + err.Param()
	case "lt":
		return "must be less than " + err.Param()
	case "uuid":
		return "must be a valid UUID"
	case "password":
		return "invalid password format"
	default:
		return "failed validation on '" + err.Tag() + "'"
	}
}
