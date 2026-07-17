package secretinput

import (
	"strings"
	"testing"
)

func TestResolve_Tier1_EnvVar(t *testing.T) {
	t.Setenv("TEST_SECRET_OK", "my-hmac-secret")

	secret, err := Resolve(
		"TEST_SECRET_OK", // envFlagVal
		"",               // directFlagVal
		false,            // directFlagChanged
		"--test-secret",  // flagLabel
		"test secret",    // promptLabel
		nil,              // warnFn
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(secret) != "my-hmac-secret" {
		t.Fatalf("got %q, want %q", string(secret), "my-hmac-secret")
	}
}

func TestResolve_Tier1_EnvVarNotSet(t *testing.T) {
	_, err := Resolve(
		"DEFINITELY_NOT_SET_XYZ123", // envFlagVal
		"",                          // directFlagVal
		false,                       // directFlagChanged
		"--test-secret",             // flagLabel
		"test secret",               // promptLabel
		nil,                         // warnFn
	)
	if err == nil {
		t.Fatalf("expected error for unset env var")
	}
	if !strings.Contains(err.Error(), "is not set") {
		t.Fatalf("expected 'is not set' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "hint") {
		t.Fatalf("expected hint in error message, got: %v", err)
	}
}

func TestResolve_Tier1_EnvVarEmpty(t *testing.T) {
	t.Setenv("TEST_SECRET_EMPTY", "")

	_, err := Resolve(
		"TEST_SECRET_EMPTY", // envFlagVal
		"",                  // directFlagVal
		false,               // directFlagChanged
		"--test-secret",     // flagLabel
		"test secret",       // promptLabel
		nil,                 // warnFn
	)
	if err == nil {
		t.Fatalf("expected error for empty env var")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected 'empty' in error, got: %v", err)
	}
}

func TestResolve_Tier2_DirectValue(t *testing.T) {
	var warnings []string
	warnFn := func(format string, args ...any) {
		warnings = append(warnings, format)
	}

	secret, err := Resolve(
		"",              // envFlagVal (not set)
		"direct-secret", // directFlagVal
		true,            // directFlagChanged
		"--test-secret", // flagLabel
		"test secret",   // promptLabel
		warnFn,          // warnFn
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(secret) != "direct-secret" {
		t.Fatalf("got %q, want %q", string(secret), "direct-secret")
	}
	if len(warnings) == 0 {
		t.Fatalf("expected ps-visibility warning")
	}
	if !strings.Contains(warnings[0], "visible in `ps`") {
		t.Fatalf("warning should mention ps visibility, got: %s", warnings[0])
	}
}

func TestResolve_Tier2_DirectValueEmpty(t *testing.T) {
	_, err := Resolve(
		"",              // envFlagVal
		"",              // directFlagVal
		true,            // directFlagChanged (flag was set but value is "")
		"--test-secret", // flagLabel
		"test secret",   // promptLabel
		nil,             // warnFn
	)
	if err == nil {
		t.Fatalf("expected error for empty direct value")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected 'empty' in error, got: %v", err)
	}
}

func TestResolve_Tier1_TakesPrecedenceOverTier2(t *testing.T) {
	t.Setenv("TEST_SECRET_PREC", "from-env")

	secret, err := Resolve(
		"TEST_SECRET_PREC", // envFlagVal
		"from-direct",      // directFlagVal
		true,               // directFlagChanged
		"--test-secret",    // flagLabel
		"test secret",      // promptLabel
		nil,                // warnFn
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(secret) != "from-env" {
		t.Fatalf("tier 1 should take precedence: got %q, want %q", string(secret), "from-env")
	}
}

// ---------------------------------------------------------------------------
// DIRECTIVE 1: ValidateForTemplate — reject backticks and template directives
// ---------------------------------------------------------------------------

func TestValidateForTemplate_BacktickRejected(t *testing.T) {
	err := ValidateForTemplate([]byte("secret`with`backticks"))
	if err == nil {
		t.Fatal("expected error for backtick in secret")
	}
	if !strings.Contains(err.Error(), "unsafe for implant template embedding") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidateForTemplate_TemplateOpenRejected(t *testing.T) {
	err := ValidateForTemplate([]byte("secret{{.Bad}}value"))
	if err == nil {
		t.Fatal("expected error for {{ in secret")
	}
	if !strings.Contains(err.Error(), "unsafe for implant template embedding") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidateForTemplate_TemplateCloseRejected(t *testing.T) {
	err := ValidateForTemplate([]byte("secret}}value"))
	if err == nil {
		t.Fatal("expected error for }} in secret")
	}
	if !strings.Contains(err.Error(), "unsafe for implant template embedding") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidateForTemplate_CleanSecretAccepted(t *testing.T) {
	err := ValidateForTemplate([]byte("perfectly-fine-secret-1234!@#$%"))
	if err != nil {
		t.Fatalf("unexpected error for clean secret: %v", err)
	}
}

// ---------------------------------------------------------------------------
// DIRECTIVE 2: whitespace preservation — secrets with leading/trailing spaces
// ---------------------------------------------------------------------------

func TestResolve_Tier1_PreservesWhitespace(t *testing.T) {
	t.Setenv("TEST_SECRET_SPACES", "  spaced-secret  ")

	secret, err := Resolve(
		"TEST_SECRET_SPACES", // envFlagVal
		"",                   // directFlagVal
		false,                // directFlagChanged
		"--test-secret",      // flagLabel
		"test secret",        // promptLabel
		nil,                  // warnFn
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(secret) != "  spaced-secret  " {
		t.Fatalf("tier 1 should preserve whitespace: got %q, want %q", string(secret), "  spaced-secret  ")
	}
}

func TestResolve_Tier2_PreservesWhitespace(t *testing.T) {
	secret, err := Resolve(
		"",                    // envFlagVal
		"  spaced-direct  ",  // directFlagVal (has leading/trailing spaces)
		true,                  // directFlagChanged
		"--test-secret",       // flagLabel
		"test secret",         // promptLabel
		func(string, ...any) {}, // warnFn (swallow warning)
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(secret) != "  spaced-direct  " {
		t.Fatalf("tier 2 should preserve whitespace: got %q, want %q", string(secret), "  spaced-direct  ")
	}
}
