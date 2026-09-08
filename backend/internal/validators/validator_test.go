package validators

import "testing"

type mobileHolder struct {
	Mobile string `json:"mobile" validate:"required,mobile"`
}

type emailHolder struct {
	Email string `json:"email" validate:"required,email"`
	Name  string `json:"name" validate:"required,min=2,max=10"`
}

func TestMobileValidation(t *testing.T) {
	v := New()

	valid := []string{"9876543210", "+919876543210", "919876543210", "09876543210", "6000000000"}
	for _, number := range valid {
		if err := v.Validate(&mobileHolder{Mobile: number}); err != nil {
			t.Errorf("%q was rejected but is a valid Indian mobile number", number)
		}
	}

	invalid := []string{
		"",            // required
		"12345",       // too short
		"5876543210",  // Indian mobile numbers do not start below 6
		"98765abcde",  // not numeric
		"98765432101", // too long
	}
	for _, number := range invalid {
		if err := v.Validate(&mobileHolder{Mobile: number}); err == nil {
			t.Errorf("%q was accepted but is not a valid mobile number", number)
		}
	}
}

func TestNormalizeMobileProducesOneCanonicalForm(t *testing.T) {
	cases := map[string]string{
		"9876543210":         "9876543210",
		"+919876543210":      "9876543210",
		"919876543210":       "9876543210",
		"09876543210":        "9876543210",
		"98765 43210":        "9876543210",
		"98765-43210":        "9876543210",
		" +91 98765-432 10 ": "9876543210",
	}
	for input, want := range cases {
		if got := NormalizeMobile(input); got != want {
			t.Errorf("NormalizeMobile(%q) = %q, want %q", input, got, want)
		}
	}
}

// Field errors are what the API returns to clients, so they must name the JSON
// field rather than the Go struct field.
func TestErrorsUseJSONFieldNames(t *testing.T) {
	v := New()
	err := v.Validate(&emailHolder{Email: "not-an-email", Name: "x"})
	if err == nil {
		t.Fatal("expected validation to fail")
	}

	fields := Errors(err)
	if len(fields) != 2 {
		t.Fatalf("got %d field errors, want 2: %+v", len(fields), fields)
	}

	byField := map[string]string{}
	for _, item := range fields {
		byField[item.Field] = item.Message
	}
	if byField["email"] != "Invalid email address" {
		t.Errorf("email message = %q", byField["email"])
	}
	if byField["name"] != "Must be at least 2 characters" {
		t.Errorf("name message = %q", byField["name"])
	}
}

func TestErrorsIgnoresNonValidationErrors(t *testing.T) {
	if fields := Errors(nil); fields != nil {
		t.Errorf("Errors(nil) = %v, want nil", fields)
	}
}
