package validator

import (
	"errors"
	"strings"
	"testing"
)

type accountInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,password"`
	Age      int    `json:"age" validate:"gte=18"`
}

func TestCustomValidatorValidateSuccess(t *testing.T) {
	t.Parallel()

	v := NewValidator()
	input := accountInput{
		Email:    "user@example.com",
		Password: "Password1",
		Age:      18,
	}

	if err := v.Validate(input); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestCustomValidatorValidateReturnsFieldErrors(t *testing.T) {
	t.Parallel()

	v := NewValidator()
	input := accountInput{
		Email:    "not-email",
		Password: "short",
		Age:      17,
	}

	err := v.Validate(input)
	if err == nil {
		t.Fatal("Validate() error = nil, want validation error")
	}

	validationErr, ok := AsValidationErr(err)
	if !ok {
		t.Fatalf("AsValidationErr() ok = false for error %T", err)
	}

	wantFields := FieldErrors{
		"email":    "invalid email format",
		"password": "invalid password format",
		"age":      "must be greater than or equal to 18",
	}
	for field, want := range wantFields {
		if got := validationErr.Fields[field]; got != want {
			t.Fatalf("Fields[%q] = %q, want %q; all fields: %#v", field, got, want, validationErr.Fields)
		}
	}
}

func TestCustomValidatorValidateField(t *testing.T) {
	t.Parallel()

	v := NewValidator()

	tests := []struct {
		name      string
		value     any
		tag       string
		wantErr   bool
		wantField string
		wantMsg   string
	}{
		{
			name:    "valid email",
			value:   "user@example.com",
			tag:     "required,email",
			wantErr: false,
		},
		{
			name:      "invalid email",
			value:     "invalid",
			tag:       "email",
			wantErr:   true,
			wantField: "value",
			wantMsg:   "invalid email format",
		},
		{
			name:      "min",
			value:     "ab",
			tag:       "min=3",
			wantErr:   true,
			wantField: "value",
			wantMsg:   "min length 3",
		},
		{
			name:      "max",
			value:     "abcd",
			tag:       "max=3",
			wantErr:   true,
			wantField: "value",
			wantMsg:   "max length 3",
		},
		{
			name:      "len",
			value:     "abcd",
			tag:       "len=3",
			wantErr:   true,
			wantField: "value",
			wantMsg:   "must be exactly 3 characters long",
		},
		{
			name:      "oneof",
			value:     "archived",
			tag:       "oneof=active done",
			wantErr:   true,
			wantField: "value",
			wantMsg:   "must be one of [active done]",
		},
		{
			name:      "gt",
			value:     1,
			tag:       "gt=1",
			wantErr:   true,
			wantField: "value",
			wantMsg:   "must be greater than 1",
		},
		{
			name:      "lte",
			value:     2,
			tag:       "lte=1",
			wantErr:   true,
			wantField: "value",
			wantMsg:   "must be less than or equal to 1",
		},
		{
			name:      "lt",
			value:     2,
			tag:       "lt=2",
			wantErr:   true,
			wantField: "value",
			wantMsg:   "must be less than 2",
		},
		{
			name:      "uuid",
			value:     "not-uuid",
			tag:       "uuid",
			wantErr:   true,
			wantField: "value",
			wantMsg:   "must be a valid UUID",
		},
		{
			name:      "password",
			value:     "nouppercase1",
			tag:       "password",
			wantErr:   true,
			wantField: "value",
			wantMsg:   "invalid password format",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := v.ValidateField(tt.value, tt.tag)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("ValidateField() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("ValidateField() error = nil, want validation error")
			}

			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("ValidateField() error type = %T, want *ValidationError", err)
			}
			if got := validationErr.Fields[tt.wantField]; got != tt.wantMsg {
				t.Fatalf("Fields[%q] = %q, want %q; all fields: %#v", tt.wantField, got, tt.wantMsg, validationErr.Fields)
			}
		})
	}
}

func TestCustomValidatorUsesJSONNamesAndDefaultMessages(t *testing.T) {
	t.Parallel()

	type input struct {
		Ignored string `json:"-" validate:"required"`
		Title   string `json:"title,omitempty" validate:"startswith=todo"`
	}

	err := NewValidator().Validate(input{Title: "task"})
	if err == nil {
		t.Fatal("Validate() error = nil, want validation errors")
	}

	validationErr, ok := AsValidationErr(err)
	if !ok {
		t.Fatalf("AsValidationErr() ok = false for %T", err)
	}
	if got := validationErr.Fields["title"]; got != "failed validation on 'startswith'" {
		t.Fatalf("title error = %q, want default startswith message; fields=%#v", got, validationErr.Fields)
	}
	if got := validationErr.Fields["Ignored"]; got != "required field" {
		t.Fatalf("json '-' field error = %q, want Go field name fallback; fields=%#v", got, validationErr.Fields)
	}
}

func TestCustomValidatorValidateRecoversFromPanic(t *testing.T) {
	t.Parallel()

	v := NewValidator()

	err := v.ValidateField("value", "unknown_validation_tag")
	if err == nil {
		t.Fatal("ValidateField() error = nil, want panic recovery error")
	}
	if !strings.Contains(err.Error(), "validator panic") {
		t.Fatalf("ValidateField() error = %q, want panic recovery error", err.Error())
	}
}

func TestValidationErrorError(t *testing.T) {
	t.Parallel()

	err := &ValidationError{Fields: FieldErrors{"field": "message"}}
	if got := err.Error(); got != "validation error" {
		t.Fatalf("Error() = %q, want %q", got, "validation error")
	}
}

func TestAsValidationErrRejectsPlainError(t *testing.T) {
	t.Parallel()

	validationErr, ok := AsValidationErr(errors.New("plain"))
	if ok {
		t.Fatalf("AsValidationErr() ok = true, want false; error = %#v", validationErr)
	}
}
