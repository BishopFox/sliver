package generate

/*
	Tests for parseTriggerLifecycleFlags — the operator-side parser
	for the Phase 2 implant lifecycle flags. Pure-Go, no console / RPC.
*/

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// newTestCmd attaches the same flags coreImplantFlags adds (for the
// lifecycle subset only — we don't need every flag for these tests).
func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	f := cmd.Flags()
	f.String("ttl", "", "")
	f.StringSlice("ttl-burn-extra-path", nil, "")
	f.StringSlice("ttl-burn-persistence", nil, "")
	f.String("trigger-wake-bind", "", "")
	f.String("trigger-wake-secret-env", "", "")
	f.String("trigger-wake-secret", "", "")
	f.StringSlice("trigger-wake-allowed-client", nil, "")
	return cmd
}

func mustSet(t *testing.T, f *pflag.FlagSet, name, value string) {
	t.Helper()
	if err := f.Set(name, value); err != nil {
		t.Fatalf("flag set %s=%q: %v", name, value, err)
	}
}

func TestParseTriggerLifecycleFlags_AllOff(t *testing.T) {
	cmd := newTestCmd()
	out, err := parseTriggerLifecycleFlags(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ttlEnabled || out.includeTriggerWake || out.ttlMinutes != 0 ||
		out.triggerWakeBindAddr != "" || out.triggerWakeSecret != nil ||
		len(out.burnExtraPaths) != 0 || len(out.burnPersistence) != 0 ||
		len(out.triggerWakeAllowedClientIDs) != 0 {
		t.Fatalf("expected zero-value struct, got %+v", out)
	}
}

func TestParseTriggerLifecycleFlags_TTLHappyPath(t *testing.T) {
	cmd := newTestCmd()
	mustSet(t, cmd.Flags(), "ttl", "720h") // 30 days
	mustSet(t, cmd.Flags(), "ttl-burn-extra-path", "/tmp/foo")
	mustSet(t, cmd.Flags(), "ttl-burn-extra-path", "/tmp/bar")
	mustSet(t, cmd.Flags(), "ttl-burn-persistence", "/etc/systemd/system/x.service")

	out, err := parseTriggerLifecycleFlags(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.ttlEnabled {
		t.Fatalf("ttlEnabled = false, want true")
	}
	if got, want := out.ttlMinutes, uint32(720*60); got != want {
		t.Fatalf("ttlMinutes = %d, want %d", got, want)
	}
	if got, want := strings.Join(out.burnExtraPaths, ","), "/tmp/foo,/tmp/bar"; got != want {
		t.Fatalf("burnExtraPaths = %q, want %q", got, want)
	}
	if got, want := strings.Join(out.burnPersistence, ","), "/etc/systemd/system/x.service"; got != want {
		t.Fatalf("burnPersistence = %q, want %q", got, want)
	}
}

func TestParseTriggerLifecycleFlags_TTLBadDuration(t *testing.T) {
	cmd := newTestCmd()
	mustSet(t, cmd.Flags(), "ttl", "30d") // Go's time.ParseDuration doesn't support 'd'
	_, err := parseTriggerLifecycleFlags(cmd)
	if err == nil {
		t.Fatalf("expected error for bad duration syntax")
	}
}

func TestParseTriggerLifecycleFlags_TTLTooSmall(t *testing.T) {
	cmd := newTestCmd()
	mustSet(t, cmd.Flags(), "ttl", "30s")
	_, err := parseTriggerLifecycleFlags(cmd)
	if err == nil || !strings.Contains(err.Error(), "at least 1 minute") {
		t.Fatalf("expected at-least-1-minute error, got %v", err)
	}
}

func TestParseTriggerLifecycleFlags_TTLNegative(t *testing.T) {
	cmd := newTestCmd()
	mustSet(t, cmd.Flags(), "ttl", "-1h")
	_, err := parseTriggerLifecycleFlags(cmd)
	if err == nil {
		t.Fatalf("expected error for non-positive duration")
	}
}

func TestParseTriggerLifecycleFlags_BurnPathsWithoutTTL(t *testing.T) {
	// Burn paths apply to both TTL-fired and operator-fired self-destruct,
	// so they're allowed without --ttl.
	cmd := newTestCmd()
	mustSet(t, cmd.Flags(), "ttl-burn-extra-path", "/tmp/x")
	out, err := parseTriggerLifecycleFlags(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ttlEnabled {
		t.Fatalf("ttlEnabled should be false")
	}
	if len(out.burnExtraPaths) != 1 || out.burnExtraPaths[0] != "/tmp/x" {
		t.Fatalf("burnExtraPaths = %v, want [/tmp/x]", out.burnExtraPaths)
	}
}

func TestParseTriggerLifecycleFlags_TriggerWakeHappyPath(t *testing.T) {
	t.Setenv("TRIGGERWAKE_SECRET_TEST", "deadbeef-secret-32-bytes-or-more")

	cmd := newTestCmd()
	mustSet(t, cmd.Flags(), "trigger-wake-bind", "0.0.0.0:46290")
	mustSet(t, cmd.Flags(), "trigger-wake-secret-env", "TRIGGERWAKE_SECRET_TEST")
	mustSet(t, cmd.Flags(), "trigger-wake-allowed-client", "ops-alice")
	mustSet(t, cmd.Flags(), "trigger-wake-allowed-client", "ops-bob")

	out, err := parseTriggerLifecycleFlags(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.includeTriggerWake {
		t.Fatalf("includeTriggerWake = false, want true")
	}
	if out.triggerWakeBindAddr != "0.0.0.0:46290" {
		t.Fatalf("triggerWakeBindAddr = %q", out.triggerWakeBindAddr)
	}
	if string(out.triggerWakeSecret) != "deadbeef-secret-32-bytes-or-more" {
		t.Fatalf("triggerWakeSecret = %q", string(out.triggerWakeSecret))
	}
	if got, want := strings.Join(out.triggerWakeAllowedClientIDs, ","), "ops-alice,ops-bob"; got != want {
		t.Fatalf("allowed = %q, want %q", got, want)
	}
}

func TestParseTriggerLifecycleFlags_TriggerWakeDirectSecret(t *testing.T) {
	cmd := newTestCmd()
	mustSet(t, cmd.Flags(), "trigger-wake-bind", "0.0.0.0:46290")
	mustSet(t, cmd.Flags(), "trigger-wake-secret", "direct-hmac-value")

	out, err := parseTriggerLifecycleFlags(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.includeTriggerWake {
		t.Fatalf("includeTriggerWake = false, want true")
	}
	if string(out.triggerWakeSecret) != "direct-hmac-value" {
		t.Fatalf("triggerWakeSecret = %q, want %q", string(out.triggerWakeSecret), "direct-hmac-value")
	}
}

func TestParseTriggerLifecycleFlags_EnvTakesPrecedenceOverDirect(t *testing.T) {
	t.Setenv("TRIGGERWAKE_PREC_TEST", "from-env-var")
	cmd := newTestCmd()
	mustSet(t, cmd.Flags(), "trigger-wake-bind", "0.0.0.0:46290")
	mustSet(t, cmd.Flags(), "trigger-wake-secret-env", "TRIGGERWAKE_PREC_TEST")
	mustSet(t, cmd.Flags(), "trigger-wake-secret", "from-direct")

	out, err := parseTriggerLifecycleFlags(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out.triggerWakeSecret) != "from-env-var" {
		t.Fatalf("env should take precedence: got %q, want %q", string(out.triggerWakeSecret), "from-env-var")
	}
}

func TestParseTriggerLifecycleFlags_TriggerWakeSecretEnvUnset(t *testing.T) {
	cmd := newTestCmd()
	mustSet(t, cmd.Flags(), "trigger-wake-bind", "0.0.0.0:46290")
	mustSet(t, cmd.Flags(), "trigger-wake-secret-env", "PROBABLY_NOT_SET_4F8E2A")
	_, err := parseTriggerLifecycleFlags(cmd)
	if err == nil || !strings.Contains(err.Error(), "is not set") {
		t.Fatalf("expected env-not-set error, got %v", err)
	}
}

func TestParseTriggerLifecycleFlags_TriggerWakeSecretEnvEmpty(t *testing.T) {
	t.Setenv("TRIGGERWAKE_EMPTY", "")
	cmd := newTestCmd()
	mustSet(t, cmd.Flags(), "trigger-wake-bind", "0.0.0.0:46290")
	mustSet(t, cmd.Flags(), "trigger-wake-secret-env", "TRIGGERWAKE_EMPTY")
	_, err := parseTriggerLifecycleFlags(cmd)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected env-empty error, got %v", err)
	}
}

func TestParseTriggerLifecycleFlags_TriggerWakeBadBind(t *testing.T) {
	t.Setenv("TRIGGERWAKE_OK", "secret")
	cmd := newTestCmd()
	mustSet(t, cmd.Flags(), "trigger-wake-bind", "not-a-host-port")
	mustSet(t, cmd.Flags(), "trigger-wake-secret-env", "TRIGGERWAKE_OK")
	_, err := parseTriggerLifecycleFlags(cmd)
	if err == nil || !strings.Contains(err.Error(), "--trigger-wake-bind") {
		t.Fatalf("expected bind-format error, got %v", err)
	}
}

func TestParseTriggerLifecycleFlags_AllowedClientWithoutBind(t *testing.T) {
	cmd := newTestCmd()
	mustSet(t, cmd.Flags(), "trigger-wake-allowed-client", "ops-alice")
	_, err := parseTriggerLifecycleFlags(cmd)
	if err == nil || !strings.Contains(err.Error(), "require --trigger-wake-bind") {
		t.Fatalf("expected bind-required error, got %v", err)
	}
}

func TestParseTriggerLifecycleFlags_SecretEnvWithoutBind(t *testing.T) {
	t.Setenv("TRIGGERWAKE_NOBIND", "some-secret")
	cmd := newTestCmd()
	mustSet(t, cmd.Flags(), "trigger-wake-secret-env", "TRIGGERWAKE_NOBIND")
	_, err := parseTriggerLifecycleFlags(cmd)
	if err == nil || !strings.Contains(err.Error(), "require --trigger-wake-bind") {
		t.Fatalf("expected bind-required error, got %v", err)
	}
}

func TestParseTriggerLifecycleFlags_DirectSecretWithoutBind(t *testing.T) {
	cmd := newTestCmd()
	mustSet(t, cmd.Flags(), "trigger-wake-secret", "orphan-secret")
	_, err := parseTriggerLifecycleFlags(cmd)
	if err == nil || !strings.Contains(err.Error(), "require --trigger-wake-bind") {
		t.Fatalf("expected bind-required error, got %v", err)
	}
}

// Sanity: the computed TTLMinutes value, fed into the server-side
// `now + minutes*time.Minute` formula, should give a time within one
// minute of the duration the operator originally requested.
func TestParseTriggerLifecycleFlags_TTLRoundTrip(t *testing.T) {
	cmd := newTestCmd()
	mustSet(t, cmd.Flags(), "ttl", "2h30m")
	out, err := parseTriggerLifecycleFlags(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := 2*time.Hour + 30*time.Minute
	got := time.Duration(out.ttlMinutes) * time.Minute
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Minute {
		t.Fatalf("round-trip drift %v > 1m (want=%v got=%v)", diff, want, got)
	}
}
