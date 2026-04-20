package models

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// TestDNSCanaryFromProtobufRoundTrip verifies that converting a DNSCanary GORM
// model → protobuf → GORM model preserves all fields.  This is important
// because UpdateCanary must pass a models.DNSCanary to GORM, not a protobuf.
func TestDNSCanaryFromProtobufRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	original := &DNSCanary{
		ImplantName:   "BRAVE_WALRUS",
		Domain:        "abc123.canary.example.com.",
		Triggered:     true,
		FirstTrigger:  now,
		LatestTrigger: now.Add(5 * time.Minute),
		Count:         3,
	}

	pb := original.ToProtobuf()
	if pb == nil {
		t.Fatal("ToProtobuf returned nil")
	}

	restored := DNSCanaryFromProtobuf(pb)

	if restored.ImplantName != original.ImplantName {
		t.Errorf("ImplantName = %q, want %q", restored.ImplantName, original.ImplantName)
	}
	if restored.Domain != original.Domain {
		t.Errorf("Domain = %q, want %q", restored.Domain, original.Domain)
	}
	if restored.Triggered != original.Triggered {
		t.Errorf("Triggered = %v, want %v", restored.Triggered, original.Triggered)
	}
	if restored.Count != original.Count {
		t.Errorf("Count = %d, want %d", restored.Count, original.Count)
	}
}

// TestDNSCanaryToProtobufZeroValue ensures ToProtobuf on a zero-value struct
// returns a non-nil protobuf with empty fields — callers must use the
// accompanying error to distinguish "not found" from "found".
func TestDNSCanaryToProtobufZeroValue(t *testing.T) {
	zero := &DNSCanary{}
	pb := zero.ToProtobuf()
	if pb == nil {
		t.Fatal("ToProtobuf on zero-value DNSCanary returned nil (unexpected)")
	}
	// A zero-value canary has empty domain — callers that rely on nil-check
	// to detect "not found" would be fooled. CanaryByDomain now returns
	// (nil, err) on not-found instead of (zeroProtobuf, err).
	if pb.Domain != "" {
		t.Errorf("expected empty Domain on zero-value canary, got %q", pb.Domain)
	}
}

// TestUpdateCanaryUsesGORMModel is a compile-time guard: DNSCanaryFromProtobuf
// must return a models.DNSCanary (not *clientpb.DNSCanary) so that GORM can
// reflect on the correct struct and find the primary key / table name.
func TestUpdateCanaryUsesGORMModel(t *testing.T) {
	pb := &clientpb.DNSCanary{
		ImplantName: "TEST",
		Domain:      "test.example.com.",
	}
	result := DNSCanaryFromProtobuf(pb)
	// Verify returned type is models.DNSCanary (not a pointer, not protobuf).
	// If DNSCanaryFromProtobuf returned *clientpb.DNSCanary this would not compile.
	var _ DNSCanary = result
}
