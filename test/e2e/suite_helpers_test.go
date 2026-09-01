package e2e

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
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

func TestSuiteScopeRouting(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		modes     []string
		wantScope string
		wantModes []string
		wantError bool
	}{
		{name: "comprehensive preserves modes", input: "comprehensive", modes: []string{"session", "beacon"}, wantScope: suiteScopeComprehensive, wantModes: []string{"session", "beacon"}},
		{name: "focused forces sessions", input: "rportfwd", modes: []string{"session", "beacon"}, wantScope: suiteScopeRportFwd, wantModes: []string{"session"}},
		{name: "surrounding whitespace", input: " rportfwd ", modes: []string{"beacon"}, wantScope: suiteScopeRportFwd, wantModes: []string{"session"}},
		{name: "unsupported", input: "other", wantError: true},
		{name: "noncanonical case", input: "RPORTFWD", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scope, err := normalizeSuiteScope(test.input)
			if test.wantError {
				if err == nil {
					t.Fatalf("normalizeSuiteScope(%q) succeeded", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeSuiteScope(%q): %v", test.input, err)
			}
			if scope != test.wantScope {
				t.Fatalf("scope = %q, want %q", scope, test.wantScope)
			}
			if got := modesForSuiteScope(scope, test.modes); !reflect.DeepEqual(got, test.wantModes) {
				t.Fatalf("modes = %v, want %v", got, test.wantModes)
			}
		})
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

func TestCommandFailureDiagnosticsReportsRunningProcessAndLogTails(t *testing.T) {
	testDir := t.TempDir()
	implantLog := filepath.Join(testDir, "implant.log")
	serverLog := filepath.Join(testDir, "server.log")
	if err := os.WriteFile(implantLog, []byte("implant-tail-marker\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(serverLog, []byte("server-tail-marker\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	process := &managedProcess{done: make(chan struct{})}
	diagnostics := commandFailureDiagnostics(process, implantLog, serverLog)
	for _, want := range []string{
		"implant process status: remains running",
		"implant-tail-marker",
		"server-tail-marker",
	} {
		if !strings.Contains(diagnostics, want) {
			t.Errorf("diagnostics missing %q", want)
		}
	}
}

func TestManagedProcessStatusReportsExactExitError(t *testing.T) {
	done := make(chan struct{})
	process := &managedProcess{done: done, err: errors.New("forced implant failure")}
	close(done)

	if got, want := managedProcessStatus(process), "exited with error: forced implant failure"; got != want {
		t.Fatalf("process status got %q, want %q", got, want)
	}
}

func TestManagedProcessStopIsIdempotent(t *testing.T) {
	const helperEnvironment = "SLIVER_E2E_MANAGED_PROCESS_HELPER"
	if os.Getenv(helperEnvironment) == "1" {
		time.Sleep(time.Hour)
		return
	}

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve test executable: %v", err)
	}
	testDir := t.TempDir()
	process, err := startProcess(
		executable,
		[]string{"-test.run=^TestManagedProcessStopIsIdempotent$"},
		testDir,
		append(os.Environ(), helperEnvironment+"=1"),
		filepath.Join(testDir, "managed-process.log"),
	)
	if err != nil {
		t.Fatalf("start managed helper process: %v", err)
	}
	if err := process.stop(); err != nil {
		t.Fatalf("first stop: %v", err)
	}
	if err := process.stop(); err != nil {
		t.Fatalf("second stop: %v", err)
	}
}

func TestReadLogTailBytesIsBoundedAndReturnsSuffix(t *testing.T) {
	const limit = int64(64)
	path := filepath.Join(t.TempDir(), "bounded.log")
	content := "discarded-prefix-" + strings.Repeat("x", int(limit)) + "-tail-marker"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	tail := readLogTailBytes(path, limit)
	if int64(len(tail)) > limit {
		t.Fatalf("log tail length got %d, want <= %d", len(tail), limit)
	}
	if strings.Contains(tail, "discarded-prefix") {
		t.Fatal("log tail included content before its byte limit")
	}
	if !strings.HasSuffix(tail, "-tail-marker") {
		t.Fatalf("log tail got %q, want suffix marker", tail)
	}
}
