package exec

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestParseExecuteShellcodeFlagsDefaults(t *testing.T) {
	cmd := newExecuteShellcodeFlagsTestCommand(t)

	cfg, changed, err := parseExecuteShellcodeFlags(cmd)
	if err != nil {
		t.Fatalf("parseExecuteShellcodeFlags returned error: %v", err)
	}
	if changed {
		t.Fatalf("expected changed=false with defaults")
	}
	if cfg == nil {
		t.Fatalf("expected non-nil config")
	}
	if cfg.Entropy != 1 || cfg.Compress != 1 || cfg.ExitOpt != 1 || cfg.Bypass != 3 || cfg.Headers != 1 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.Thread || cfg.Unicode || cfg.OEP != 0 {
		t.Fatalf("unexpected default bool/OEP values: %+v", cfg)
	}
}

func TestParseExecuteShellcodeFlagsChanged(t *testing.T) {
	cmd := newExecuteShellcodeFlagsTestCommand(t)
	if err := cmd.Flags().Set("shellcode-entropy", "2"); err != nil {
		t.Fatalf("set shellcode-entropy: %v", err)
	}
	if err := cmd.Flags().Set("shellcode-compress", "true"); err != nil {
		t.Fatalf("set shellcode-compress: %v", err)
	}
	if err := cmd.Flags().Set("shellcode-thread", "true"); err != nil {
		t.Fatalf("set shellcode-thread: %v", err)
	}

	cfg, changed, err := parseExecuteShellcodeFlags(cmd)
	if err != nil {
		t.Fatalf("parseExecuteShellcodeFlags returned error: %v", err)
	}
	if !changed {
		t.Fatalf("expected changed=true")
	}
	if cfg.Entropy != 2 || cfg.Compress != 2 || !cfg.Thread {
		t.Fatalf("unexpected changed config: %+v", cfg)
	}
}

func TestParseExecuteShellcodeFlagsValidation(t *testing.T) {
	cmd := newExecuteShellcodeFlagsTestCommand(t)
	if err := cmd.Flags().Set("shellcode-bypass", "9"); err != nil {
		t.Fatalf("set shellcode-bypass: %v", err)
	}
	if _, _, err := parseExecuteShellcodeFlags(cmd); err == nil {
		t.Fatalf("expected validation error for shellcode-bypass")
	}
}

func TestShouldConvertExecuteShellcodePE(t *testing.T) {
	testCases := []struct {
		name         string
		path         string
		flagsChanged bool
		wantConvert  bool
		wantDLL      bool
	}{
		{name: "exe by extension", path: "/tmp/payload.exe", flagsChanged: false, wantConvert: true, wantDLL: false},
		{name: "dll by extension", path: "/tmp/payload.dll", flagsChanged: false, wantConvert: true, wantDLL: true},
		{name: "raw shellcode", path: "/tmp/payload.bin", flagsChanged: false, wantConvert: false, wantDLL: false},
		{name: "flags force conversion", path: "/tmp/payload.bin", flagsChanged: true, wantConvert: true, wantDLL: false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotConvert, gotDLL := shouldConvertExecuteShellcodePE(tc.path, tc.flagsChanged)
			if gotConvert != tc.wantConvert || gotDLL != tc.wantDLL {
				t.Fatalf("unexpected result convert=%v dll=%v, expected convert=%v dll=%v", gotConvert, gotDLL, tc.wantConvert, tc.wantDLL)
			}
		})
	}
}

func newExecuteShellcodeFlagsTestCommand(t *testing.T) *cobra.Command {
	t.Helper()

	cmd := &cobra.Command{Use: "execute-shellcode"}
	cmd.Flags().Uint32("shellcode-entropy", 1, "")
	cmd.Flags().Bool("shellcode-compress", false, "")
	cmd.Flags().Uint32("shellcode-exitopt", 1, "")
	cmd.Flags().Uint32("shellcode-bypass", 3, "")
	cmd.Flags().Uint32("shellcode-headers", 1, "")
	cmd.Flags().Bool("shellcode-thread", false, "")
	cmd.Flags().Bool("shellcode-unicode", false, "")
	cmd.Flags().Uint32("shellcode-oep", 0, "")
	return cmd
}
