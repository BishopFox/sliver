package e2e

import (
	"testing"

	"github.com/bishopfox/sliver/test/e2e/shellcodecoverage"
)

func TestShellcodeExecutionProtection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		arch    string
		encoder string
		want    string
	}{
		{name: "amd64 raw", arch: "amd64", encoder: shellcodecoverage.EncoderNone, want: "rx"},
		{name: "amd64 SGN", arch: "amd64", encoder: shellcodecoverage.EncoderShikataGaNai, want: "rwx"},
		{name: "amd64 XOR", arch: "amd64", encoder: shellcodecoverage.EncoderXOR, want: "rwx"},
		{name: "386 raw", arch: "386", encoder: shellcodecoverage.EncoderNone, want: "rx"},
		{name: "386 SGN", arch: "386", encoder: shellcodecoverage.EncoderShikataGaNai, want: "rwx"},
		{name: "arm64 raw", arch: "arm64", encoder: shellcodecoverage.EncoderNone, want: "rx"},
		{name: "arm64 XOR", arch: "arm64", encoder: shellcodecoverage.EncoderXOR, want: "rx"},
		{name: "arm64 dynamic XOR", arch: "arm64", encoder: shellcodecoverage.EncoderXORDynamic, want: "rx"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := shellcodeExecutionProtection(test.arch, test.encoder); got != test.want {
				t.Fatalf("shellcodeExecutionProtection(%q, %q) = %q, want %q", test.arch, test.encoder, got, test.want)
			}
		})
	}
}
