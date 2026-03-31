package err

import (
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
)

type testStruct struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required,min=8,max=64"`
}

func mustValidationErrors(t *testing.T, s testStruct) validator.ValidationErrors {
	t.Helper()
	validate := validator.New()
	err := validate.Struct(s)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		t.Fatalf("expected validator.ValidationErrors, got %T", err)
	}
	return ve
}

func findByField(results []ValidationFieldError, field string) *ValidationFieldError {
	for i := range results {
		if results[i].Field == field {
			return &results[i]
		}
	}
	return nil
}

func TestTranslateValidationErrors_RequiredTag(t *testing.T) {
	ve := mustValidationErrors(t, testStruct{Email: "", Password: ""})
	result := TranslateValidationErrors(ve)

	emailErr := findByField(result, "email")
	if emailErr == nil {
		t.Fatal("expected error for email field")
		return
	}
	if emailErr.Message != "email is required" {
		t.Fatalf("expected message %q, got %q", "email is required", emailErr.Message)
	}
}

func TestTranslateValidationErrors_EmailTag(t *testing.T) {
	ve := mustValidationErrors(t, testStruct{Email: "not-an-email", Password: "longpassword123"})
	result := TranslateValidationErrors(ve)

	emailErr := findByField(result, "email")
	if emailErr == nil {
		t.Fatal("expected error for email field")
		return
	}
	if emailErr.Message != "email must be a valid email" {
		t.Fatalf("expected message %q, got %q", "email must be a valid email", emailErr.Message)
	}
}

func TestTranslateValidationErrors_MinTag(t *testing.T) {
	ve := mustValidationErrors(t, testStruct{Email: "a@b.com", Password: "short"})
	result := TranslateValidationErrors(ve)

	pwErr := findByField(result, "password")
	if pwErr == nil {
		t.Fatal("expected error for password field")
		return
	}
	if pwErr.Message != "password must be at least 8" {
		t.Fatalf("expected message %q, got %q", "password must be at least 8", pwErr.Message)
	}
}

func TestTranslateValidationErrors_MaxTag(t *testing.T) {
	type maxStruct struct {
		Name string `validate:"max=3"`
	}
	validate := validator.New()
	err := validate.Struct(maxStruct{Name: "toolong"})
	if err == nil {
		t.Fatal("expected validation error")
	}
	result := TranslateValidationErrors(err)

	nameErr := findByField(result, "name")
	if nameErr == nil {
		t.Fatal("expected error for name field")
		return
	}
	if nameErr.Message != "name must be at most 3" {
		t.Fatalf("expected message %q, got %q", "name must be at most 3", nameErr.Message)
	}
}

func TestTranslateValidationErrors_FieldNameIsLowercased(t *testing.T) {
	ve := mustValidationErrors(t, testStruct{Email: "", Password: ""})
	result := TranslateValidationErrors(ve)

	for _, fe := range result {
		// Struct fields are "Email" and "Password"; translated fields must be lowercase
		if fe.Field != "email" && fe.Field != "password" {
			t.Fatalf("expected lowercase field name, got %q", fe.Field)
		}
	}
}

func TestTranslateValidationErrors_RejectedValueCaptured(t *testing.T) {
	ve := mustValidationErrors(t, testStruct{Email: "bad-email", Password: "short"})
	result := TranslateValidationErrors(ve)

	emailErr := findByField(result, "email")
	if emailErr == nil {
		t.Fatal("expected error for email field")
		return
	}
	if emailErr.RejectedValue != "bad-email" {
		t.Fatalf("expected rejected value %q, got %v", "bad-email", emailErr.RejectedValue)
	}

	pwErr := findByField(result, "password")
	if pwErr == nil {
		t.Fatal("expected error for password field")
		return
	}
	if pwErr.RejectedValue != "short" {
		t.Fatalf("expected rejected value %q, got %v", "short", pwErr.RejectedValue)
	}
}

func TestTranslateValidationErrors_NonValidatorErrorReturnsNil(t *testing.T) {
	plainErr := errors.New("some random error")
	result := TranslateValidationErrors(plainErr)

	if result != nil {
		t.Fatalf("expected nil for non-validator error, got %v", result)
	}
}
