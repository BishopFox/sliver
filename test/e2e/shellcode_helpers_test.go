package e2e

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

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

func TestShellcodeEncoderSamples(t *testing.T) {
	t.Parallel()

	if got := shellcodeEncoderSamples(shellcodecoverage.EncoderShikataGaNai, 4); got != 4 {
		t.Fatalf("SGN samples = %d, want 4", got)
	}
	for _, encoder := range []string{
		shellcodecoverage.EncoderNone,
		shellcodecoverage.EncoderXOR,
		shellcodecoverage.EncoderXORDynamic,
	} {
		if got := shellcodeEncoderSamples(encoder, 4); got != 1 {
			t.Fatalf("%s samples = %d, want 1", encoder, got)
		}
	}
}

func TestRunShellcodeSamplesSuccess(t *testing.T) {
	t.Parallel()

	encodeCalls := 0
	executeCalls := 0
	verifiedCalls := 0
	result, err := runShellcodeSamples(
		[]byte{0xaa},
		4,
		func(_ []byte) ([]byte, error) {
			encodeCalls++
			payload := make([]byte, encodeCalls+2)
			for index := range payload {
				payload[index] = byte(encodeCalls)
			}
			return payload, nil
		},
		func(payload []byte, sample int, phase shellcodeExecutionPhase) error {
			executeCalls++
			if phase != shellcodeExecutionPrimary {
				t.Fatalf("execute phase = %q, want %q", phase, shellcodeExecutionPrimary)
			}
			if sample != executeCalls {
				t.Fatalf("execute sample = %d, want %d", sample, executeCalls)
			}
			if got, want := len(payload), sample+2; got != want {
				t.Fatalf("execute payload bytes = %d, want %d", got, want)
			}
			return nil
		},
		func(sample int, payload []byte, _ [32]byte) {
			verifiedCalls++
			if sample != verifiedCalls {
				t.Fatalf("verified sample = %d, want %d", sample, verifiedCalls)
			}
			if got, want := len(payload), sample+2; got != want {
				t.Fatalf("verified payload bytes = %d, want %d", got, want)
			}
		},
	)
	if err != nil {
		t.Fatalf("runShellcodeSamples() error = %v", err)
	}
	if encodeCalls != 4 || executeCalls != 4 || verifiedCalls != 4 {
		t.Fatalf("callback calls = encode %d, execute %d, verified %d; want 4 each", encodeCalls, executeCalls, verifiedCalls)
	}
	if result.completedSamples != 4 || result.payloadBytes != 6 {
		t.Fatalf("result = %+v, want 4 completed samples and 6 payload bytes", result)
	}
}

func TestRunShellcodeSamplesStopsOnFirstEncodeFailure(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("encode failed")
	encodeCalls := 0
	executeCalls := 0
	result, err := runShellcodeSamples(
		[]byte{0xaa},
		4,
		func(_ []byte) ([]byte, error) {
			encodeCalls++
			return nil, wantErr
		},
		func(_ []byte, _ int, _ shellcodeExecutionPhase) error {
			executeCalls++
			return nil
		},
		nil,
	)
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "sample 1/4 ShellcodeEncoder RPC failed") {
		t.Fatalf("runShellcodeSamples() error = %v, want first-sample encode diagnostic", err)
	}
	if encodeCalls != 1 || executeCalls != 0 {
		t.Fatalf("callback calls = encode %d, execute %d; want 1 and 0", encodeCalls, executeCalls)
	}
	if result.completedSamples != 0 || result.payloadBytes != 0 {
		t.Fatalf("result = %+v, want zero completed samples and payload bytes", result)
	}
}

func TestRunShellcodeSamplesDiagnosticReplayCannotTurnFailureGreen(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("execute failed")
	encodeCalls := 0
	executeCalls := 0
	verifiedCalls := 0
	wantPayload := []byte{1, 2, 3, 4, 5}
	var phases []shellcodeExecutionPhase
	result, err := runShellcodeSamples(
		[]byte{0xaa},
		4,
		func(_ []byte) ([]byte, error) {
			encodeCalls++
			return append([]byte(nil), wantPayload...), nil
		},
		func(payload []byte, sample int, phase shellcodeExecutionPhase) error {
			executeCalls++
			phases = append(phases, phase)
			if sample != 1 {
				t.Fatalf("execute sample = %d, want 1", sample)
			}
			if !bytes.Equal(payload, wantPayload) {
				t.Fatalf("execute payload = %x, want %x", payload, wantPayload)
			}
			if phase == shellcodeExecutionPrimary {
				return wantErr
			}
			return nil
		},
		func(_ int, _ []byte, _ [32]byte) {
			verifiedCalls++
		},
	)
	if !errors.Is(err, wantErr) ||
		!strings.Contains(err.Error(), "sample 1/4 payload sha256=") ||
		!strings.Contains(err.Error(), "same-byte diagnostic replay succeeded; original failure remains authoritative") {
		t.Fatalf("runShellcodeSamples() error = %v, want authoritative primary failure and successful replay diagnostic", err)
	}
	if encodeCalls != 1 || executeCalls != 2 || verifiedCalls != 0 {
		t.Fatalf("callback calls = encode %d, execute %d, verified %d; want 1, 2, and 0", encodeCalls, executeCalls, verifiedCalls)
	}
	wantPhases := []shellcodeExecutionPhase{shellcodeExecutionPrimary, shellcodeExecutionDiagnosticReplay}
	if len(phases) != len(wantPhases) || phases[0] != wantPhases[0] || phases[1] != wantPhases[1] {
		t.Fatalf("execution phases = %v, want %v", phases, wantPhases)
	}
	if result.completedSamples != 0 || result.payloadBytes != 5 {
		t.Fatalf("result = %+v, want zero completed samples and 5 payload bytes", result)
	}
}

func TestRunShellcodeSamplesReportsDiagnosticReplayFailure(t *testing.T) {
	t.Parallel()

	primaryErr := errors.New("primary crashed")
	replayErr := errors.New("replay crashed")
	wantPayload := []byte{5, 4, 3, 2, 1}
	var calls []shellcodeExecutionPhase
	result, err := runShellcodeSamples(
		[]byte{0xaa},
		4,
		func(_ []byte) ([]byte, error) {
			return append([]byte(nil), wantPayload...), nil
		},
		func(payload []byte, sample int, phase shellcodeExecutionPhase) error {
			calls = append(calls, phase)
			if sample != 1 {
				t.Fatalf("execute sample = %d, want 1", sample)
			}
			if !bytes.Equal(payload, wantPayload) {
				t.Fatalf("execute payload = %x, want %x", payload, wantPayload)
			}
			if phase == shellcodeExecutionPrimary {
				return primaryErr
			}
			return replayErr
		},
		nil,
	)
	if !errors.Is(err, primaryErr) || !errors.Is(err, replayErr) ||
		!strings.Contains(err.Error(), "same-byte diagnostic replay failed") {
		t.Fatalf("runShellcodeSamples() error = %v, want both primary and replay failures", err)
	}
	wantCalls := []shellcodeExecutionPhase{shellcodeExecutionPrimary, shellcodeExecutionDiagnosticReplay}
	if len(calls) != len(wantCalls) || calls[0] != wantCalls[0] || calls[1] != wantCalls[1] {
		t.Fatalf("execution phases = %v, want %v", calls, wantCalls)
	}
	if result.completedSamples != 0 || result.payloadBytes != int64(len(wantPayload)) {
		t.Fatalf("result = %+v, want zero completed samples and %d payload bytes", result, len(wantPayload))
	}
}

func TestRunShellcodeSamplesRejectsDuplicateOutput(t *testing.T) {
	t.Parallel()

	encodeCalls := 0
	executeCalls := 0
	result, err := runShellcodeSamples(
		[]byte{0xaa},
		4,
		func(_ []byte) ([]byte, error) {
			encodeCalls++
			return []byte{1, 2, 3, 4}, nil
		},
		func(_ []byte, _ int, phase shellcodeExecutionPhase) error {
			executeCalls++
			if phase != shellcodeExecutionPrimary {
				t.Fatalf("execute phase = %q, want %q", phase, shellcodeExecutionPrimary)
			}
			return nil
		},
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "sample 2/4 duplicated sample 1 payload sha256=") {
		t.Fatalf("runShellcodeSamples() error = %v, want duplicate-output diagnostic", err)
	}
	if encodeCalls != 2 || executeCalls != 1 {
		t.Fatalf("callback calls = encode %d, execute %d; want 2 and 1", encodeCalls, executeCalls)
	}
	if result.completedSamples != 1 || result.payloadBytes != 4 {
		t.Fatalf("result = %+v, want one completed sample and 4 payload bytes", result)
	}
}

func TestShellcodeFailureDetailPreservesValidUTF8WhenTruncated(t *testing.T) {
	t.Parallel()

	prefix := strings.Repeat("a", shellcodeFailureDetailBytes-1)
	detail := prefix + "é-tail"
	got := shellcodeFailureDetail(detail)
	want := prefix + "...(truncated)"
	if got != want {
		t.Fatalf("shellcodeFailureDetail() produced an unexpected boundary: got suffix %q, want %q", got[len(got)-32:], want[len(want)-32:])
	}
	if !utf8.ValidString(got) {
		t.Fatal("shellcodeFailureDetail() returned invalid UTF-8 after truncation")
	}
}

func TestShellcodeFailureDetailNormalizesInvalidLogBytes(t *testing.T) {
	t.Parallel()

	got := shellcodeFailureDetail(" runner \xff log ")
	if !utf8.ValidString(got) {
		t.Fatal("shellcodeFailureDetail() returned invalid UTF-8 after normalization")
	}
	if got != "runner � log" {
		t.Fatalf("shellcodeFailureDetail() = %q, want invalid byte replaced and whitespace trimmed", got)
	}
}
