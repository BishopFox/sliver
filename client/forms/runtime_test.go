package forms

import "testing"

func TestShouldUseStdoutTTYForForm(t *testing.T) {
	tests := []struct {
		name      string
		stdinTTY  bool
		stdoutTTY bool
		stderrTTY bool
		want      bool
	}{
		{name: "stderr redirected but stdout interactive", stdinTTY: true, stdoutTTY: true, stderrTTY: false, want: true},
		{name: "default stderr already interactive", stdinTTY: true, stdoutTTY: true, stderrTTY: true, want: false},
		{name: "non interactive stdin", stdinTTY: false, stdoutTTY: true, stderrTTY: false, want: false},
		{name: "all stdio piped", stdinTTY: true, stdoutTTY: false, stderrTTY: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldUseStdoutTTYForForm(tt.stdinTTY, tt.stdoutTTY, tt.stderrTTY); got != tt.want {
				t.Fatalf("shouldUseStdoutTTYForForm(%v, %v, %v) = %v, want %v", tt.stdinTTY, tt.stdoutTTY, tt.stderrTTY, got, tt.want)
			}
		})
	}
}

func TestNeedsDedicatedTTYForForm(t *testing.T) {
	tests := []struct {
		name      string
		stdinTTY  bool
		stdoutTTY bool
		stderrTTY bool
		want      bool
	}{
		{name: "interactive stdin with piped stdout and stderr", stdinTTY: true, stdoutTTY: false, stderrTTY: false, want: true},
		{name: "stdout can be reused", stdinTTY: true, stdoutTTY: true, stderrTTY: false, want: false},
		{name: "stderr already interactive", stdinTTY: true, stdoutTTY: false, stderrTTY: true, want: false},
		{name: "non interactive stdin", stdinTTY: false, stdoutTTY: false, stderrTTY: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needsDedicatedTTYForForm(tt.stdinTTY, tt.stdoutTTY, tt.stderrTTY); got != tt.want {
				t.Fatalf("needsDedicatedTTYForForm(%v, %v, %v) = %v, want %v", tt.stdinTTY, tt.stdoutTTY, tt.stderrTTY, got, tt.want)
			}
		})
	}
}
