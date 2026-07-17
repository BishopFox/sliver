package models

/*
	Tests for the ImplantConfig trigger-wake / TTL field round-trip
	through ToProtobuf() and ImplantConfigFromProtobuf().
*/

import (
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestImplantConfig_TriggerWakeFieldsRoundTrip(t *testing.T) {
	// Start with a proto config that has all trigger wake fields set.
	pbConfig := &clientpb.ImplantConfig{
		GOOS:                        "linux",
		GOARCH:                      "amd64",
		IncludeTriggerWake:          true,
		TriggerWakeBindAddr:         "0.0.0.0:46290",
		TriggerWakeSecret:           []byte("test-hmac-secret-32-bytes-long!!"),
		TriggerWakeAllowedClientIDs: []string{"ops-alice", "ops-bob"},
		TTLEnabled:                  true,
		TTLMinutes:                  720,
		TTLExpiresAtUnix:            1735689600,
		TTLBurnExtraPaths:           []string{"/tmp/foo", "/tmp/bar"},
		TTLBurnPersistence:          []string{"/etc/systemd/system/x.service"},
	}

	// Convert proto -> native model.
	native := ImplantConfigFromProtobuf(pbConfig)

	if !native.IncludeTriggerWake {
		t.Fatalf("IncludeTriggerWake = false, want true")
	}
	if native.TriggerWakeBindAddr != "0.0.0.0:46290" {
		t.Fatalf("TriggerWakeBindAddr = %q, want %q", native.TriggerWakeBindAddr, "0.0.0.0:46290")
	}
	if string(native.TriggerWakeSecret) != "test-hmac-secret-32-bytes-long!!" {
		t.Fatalf("TriggerWakeSecret = %q, want %q", string(native.TriggerWakeSecret), "test-hmac-secret-32-bytes-long!!")
	}
	if native.TriggerWakeAllowedClientIDs != "ops-alice,ops-bob" {
		t.Fatalf("TriggerWakeAllowedClientIDs = %q, want %q", native.TriggerWakeAllowedClientIDs, "ops-alice,ops-bob")
	}
	if !native.TTLEnabled {
		t.Fatalf("TTLEnabled = false, want true")
	}
	if native.TTLMinutes != 720 {
		t.Fatalf("TTLMinutes = %d, want 720", native.TTLMinutes)
	}
	if native.TTLExpiresAtUnix != 1735689600 {
		t.Fatalf("TTLExpiresAtUnix = %d, want 1735689600", native.TTLExpiresAtUnix)
	}
	if native.TTLBurnExtraPaths != "/tmp/foo,/tmp/bar" {
		t.Fatalf("TTLBurnExtraPaths = %q, want %q", native.TTLBurnExtraPaths, "/tmp/foo,/tmp/bar")
	}
	if native.TTLBurnPersistence != "/etc/systemd/system/x.service" {
		t.Fatalf("TTLBurnPersistence = %q, want %q", native.TTLBurnPersistence, "/etc/systemd/system/x.service")
	}

	// Convert native model -> proto (round-trip).
	pbRoundTrip := native.ToProtobuf()

	if !pbRoundTrip.IncludeTriggerWake {
		t.Fatalf("round-trip IncludeTriggerWake = false")
	}
	if pbRoundTrip.TriggerWakeBindAddr != "0.0.0.0:46290" {
		t.Fatalf("round-trip TriggerWakeBindAddr = %q", pbRoundTrip.TriggerWakeBindAddr)
	}
	if string(pbRoundTrip.TriggerWakeSecret) != "test-hmac-secret-32-bytes-long!!" {
		t.Fatalf("round-trip TriggerWakeSecret = %q", string(pbRoundTrip.TriggerWakeSecret))
	}
	if got := strings.Join(pbRoundTrip.TriggerWakeAllowedClientIDs, ","); got != "ops-alice,ops-bob" {
		t.Fatalf("round-trip AllowedClientIDs = %q", got)
	}
	if !pbRoundTrip.TTLEnabled {
		t.Fatalf("round-trip TTLEnabled = false")
	}
	if pbRoundTrip.TTLMinutes != 720 {
		t.Fatalf("round-trip TTLMinutes = %d", pbRoundTrip.TTLMinutes)
	}
	if pbRoundTrip.TTLExpiresAtUnix != 1735689600 {
		t.Fatalf("round-trip TTLExpiresAtUnix = %d", pbRoundTrip.TTLExpiresAtUnix)
	}
	if got := strings.Join(pbRoundTrip.TTLBurnExtraPaths, ","); got != "/tmp/foo,/tmp/bar" {
		t.Fatalf("round-trip TTLBurnExtraPaths = %q", got)
	}
	if got := strings.Join(pbRoundTrip.TTLBurnPersistence, ","); got != "/etc/systemd/system/x.service" {
		t.Fatalf("round-trip TTLBurnPersistence = %q", got)
	}
}

func TestImplantConfig_TriggerWakeFieldsOff(t *testing.T) {
	// Trigger wake off: all fields should round-trip as zero values.
	pbConfig := &clientpb.ImplantConfig{
		GOOS:   "linux",
		GOARCH: "amd64",
	}

	native := ImplantConfigFromProtobuf(pbConfig)
	if native.IncludeTriggerWake {
		t.Fatalf("IncludeTriggerWake should be false when not set")
	}
	if native.TTLEnabled {
		t.Fatalf("TTLEnabled should be false when not set")
	}

	pbRoundTrip := native.ToProtobuf()
	if pbRoundTrip.IncludeTriggerWake {
		t.Fatalf("round-trip IncludeTriggerWake should be false")
	}
	if len(pbRoundTrip.TriggerWakeAllowedClientIDs) != 0 {
		t.Fatalf("round-trip AllowedClientIDs should be empty, got %v", pbRoundTrip.TriggerWakeAllowedClientIDs)
	}
}

func TestSplitNonEmpty(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{"a,,b", []string{"a", "b"}},
		{",", nil},
	}
	for _, tt := range tests {
		got := splitNonEmpty(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("splitNonEmpty(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i, g := range got {
			if g != tt.want[i] {
				t.Errorf("splitNonEmpty(%q)[%d] = %q, want %q", tt.input, i, g, tt.want[i])
			}
		}
	}
}
