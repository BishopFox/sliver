package e2e

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseSelectionPreservesAllowedOrderAndDeduplicates(t *testing.T) {
	allowed := []string{"mtls", "wg", "http"}
	got, err := parseSelection(" HTTP, mtls,http, WG,mtls ", allowed, "transport")
	if err != nil {
		t.Fatalf("parse selection: %v", err)
	}
	want := []string{"mtls", "wg", "http"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selection got %v, want %v", got, want)
	}
}

func TestParseSelectionRejectsUnknownsDeterministically(t *testing.T) {
	_, err := parseSelection("zeta,mtls,alpha", []string{"mtls", "wg", "http"}, "transport")
	if err == nil {
		t.Fatal("expected unknown selection error")
	}
	if got, want := err.Error(), "unknown transport selection: alpha, zeta"; got != want {
		t.Fatalf("error got %q, want %q", got, want)
	}
}

func TestParseSelectionRejectsEmptyInput(t *testing.T) {
	_, err := parseSelection(" , \t,", []string{"session", "beacon"}, "implant mode")
	if err == nil {
		t.Fatal("expected empty selection error")
	}
	if got, want := err.Error(), "at least one implant mode is required"; got != want {
		t.Fatalf("error got %q, want %q", got, want)
	}
}

func TestSanitizedImplantEnvUsesAllowlistAndIsolatedValues(t *testing.T) {
	hostEnv := []string{
		"Path=/first/bin",
		"PATH=/second/bin",
		"systemroot=/system-root",
		"ComSpec=/system-shell",
		"HOME=/host/home",
		"TMP=/host/tmp",
		"AWS_SECRET_ACCESS_KEY=do-not-copy",
		"HTTP_PROXY=http://operator-secret",
		"SLIVER_E2E_MARKER=host-marker",
		"MALFORMED",
	}
	got := sanitizedImplantEnv(hostEnv, "/isolated/home", "/isolated/temp", "public-marker")

	values := map[string]string{}
	for _, entry := range got {
		name, value, found := strings.Cut(entry, "=")
		if !found {
			t.Fatalf("environment contains malformed entry %q", entry)
		}
		upper := strings.ToUpper(name)
		if _, duplicate := values[upper]; duplicate {
			t.Fatalf("environment contains duplicate key %q: %v", name, got)
		}
		values[upper] = value
	}

	want := map[string]string{
		"PATH":                     "/second/bin",
		"SYSTEMROOT":               "/system-root",
		"COMSPEC":                  "/system-shell",
		"HOME":                     "/isolated/home",
		"USERPROFILE":              "/isolated/home",
		"TMP":                      "/isolated/temp",
		"TEMP":                     "/isolated/temp",
		"TMPDIR":                   "/isolated/temp",
		"SLIVER_E2E_PARENT_MARKER": "public-marker",
		"SLIVER_E2E_MARKER":        "public-marker",
	}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("sanitized environment got %#v, want %#v", values, want)
	}

	for _, secret := range []string{"AWS_SECRET_ACCESS_KEY", "HTTP_PROXY"} {
		if _, exists := values[secret]; exists {
			t.Fatalf("sanitized environment leaked %s", secret)
		}
	}
	for index := 1; index < len(got); index++ {
		if got[index-1] > got[index] {
			t.Fatalf("environment is not sorted: %v", got)
		}
	}
}

func TestValidateConnectedTransport(t *testing.T) {
	tests := []struct {
		name      string
		transport string
		activeC2  string
		expected  string
		wantError string
	}{
		{name: "mtls", transport: "mTLS", activeC2: "MTLS://127.0.0.1:8888", expected: "mtls"},
		{name: "wireguard", transport: "wg", activeC2: "wg://127.0.0.1:53", expected: "wg"},
		{name: "http", transport: "http(s)", activeC2: "http://127.0.0.1:8080/path", expected: "http"},
		{name: "empty active C2 allowed", transport: "mtls", expected: "mtls"},
		{name: "wrong transport", transport: "wg", activeC2: "mtls://127.0.0.1", expected: "mtls", wantError: "transport got"},
		{name: "wrong active scheme", transport: "wg", activeC2: "http://127.0.0.1", expected: "wg", wantError: "want wg scheme"},
		{name: "missing active scheme", transport: "http(s)", activeC2: "127.0.0.1:8080", expected: "http", wantError: "want http scheme"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateConnectedTransport(test.transport, test.activeC2, test.expected)
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("validate transport: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error got %v, want substring %q", err, test.wantError)
			}
		})
	}
}
