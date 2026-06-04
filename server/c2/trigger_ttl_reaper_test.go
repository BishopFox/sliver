package c2

/*
	Tests for the TTL reaper's pure-logic helpers. The full sweep()
	function depends on DB state and is exercised by integration tests;
	these unit tests cover the helper functions that don't touch the DB.
*/

import (
	"testing"
	"time"

	"github.com/bishopfox/sliver/server/db/models"
)

func TestIndexActivitiesByConfig(t *testing.T) {
	now := time.Now()
	activities := []*models.TriggerActivity{
		{ImplantConfigID: "aaa", LastSeenIP: "10.0.0.1", LastActivityAt: now},
		{ImplantConfigID: "bbb", LastSeenIP: "10.0.0.2", LastActivityAt: now.Add(-time.Hour)},
	}

	m := indexActivitiesByConfig(activities)
	if len(m) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m))
	}
	if m["aaa"].LastSeenIP != "10.0.0.1" {
		t.Errorf("aaa IP = %q, want 10.0.0.1", m["aaa"].LastSeenIP)
	}
	if m["bbb"].LastSeenIP != "10.0.0.2" {
		t.Errorf("bbb IP = %q, want 10.0.0.2", m["bbb"].LastSeenIP)
	}
	if _, ok := m["nonexistent"]; ok {
		t.Error("unexpected key 'nonexistent'")
	}
}

func TestIndexActivitiesByConfigEmpty(t *testing.T) {
	m := indexActivitiesByConfig(nil)
	if len(m) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(m))
	}
}

func TestIndexActivitiesByConfigDuplicate(t *testing.T) {
	// If there are duplicate config IDs (shouldn't happen due to unique
	// index, but be defensive), the last one wins.
	now := time.Now()
	activities := []*models.TriggerActivity{
		{ImplantConfigID: "aaa", LastSeenIP: "10.0.0.1", LastActivityAt: now},
		{ImplantConfigID: "aaa", LastSeenIP: "10.0.0.99", LastActivityAt: now},
	}
	m := indexActivitiesByConfig(activities)
	if len(m) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(m))
	}
	if m["aaa"].LastSeenIP != "10.0.0.99" {
		t.Errorf("expected last-writer-wins, got %q", m["aaa"].LastSeenIP)
	}
}

func TestReaperCooldownConstant(t *testing.T) {
	// Sanity: cooldown should be at least 30 minutes, and the default
	// interval should be reasonable.
	if reaperCooldown < 30*time.Minute {
		t.Errorf("reaperCooldown = %v, should be at least 30m", reaperCooldown)
	}
	if DefaultReaperInterval < time.Minute {
		t.Errorf("DefaultReaperInterval = %v, should be at least 1m", DefaultReaperInterval)
	}
}
