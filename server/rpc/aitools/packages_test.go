package aitools

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseAIExtensionManifestBOFExecutor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		executor  string
		dependsOn string
		wantErr   string
	}{
		{name: "reflektor", executor: bofExecutorReflektor},
		{name: "reflektor with fallback", executor: bofExecutorReflektor, dependsOn: "coff-loader"},
		{name: "coff loader", executor: bofExecutorCOFFLoader, dependsOn: "custom-loader"},
		{name: "coff loader requires dependency", executor: bofExecutorCOFFLoader, wantErr: "requires depends_on"},
		{name: "coff loader rejects whitespace dependency", executor: bofExecutorCOFFLoader, dependsOn: " \t", wantErr: "requires depends_on"},
		{name: "executor must be exact", executor: " reflektor ", wantErr: "invalid bof_executor"},
		{name: "unknown", executor: "other", wantErr: "invalid bof_executor"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := fmt.Sprintf(
				"{\"name\":\"test\",\"commands\":[{\"command_name\":\"test\",\"help\":\"test\",\"entrypoint\":\"go\",\"bof_executor\":%q,\"depends_on\":%q,\"files\":[{\"os\":\"windows\",\"arch\":\"amd64\",\"path\":\"test.o\"}]}]}",
				test.executor,
				test.dependsOn,
			)
			parsed, err := parseAIExtensionManifest([]byte(manifest))
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse manifest: %v", err)
			}
			if got := parsed.ExtCommand[0].BOFExecutor; got != test.executor {
				t.Fatalf("unexpected bof_executor %q", got)
			}
		})
	}
}
