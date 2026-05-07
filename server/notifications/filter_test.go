package notifications

import "testing"

func TestResolveEvents(t *testing.T) {
	global := []string{"a", "b"}
	service := []string{"c"}

	set := resolveEvents(global, nil)
	if len(set) != 2 {
		t.Fatalf("expected global events, got %v", set)
	}

	set = resolveEvents(global, service)
	if len(set) != 1 {
		t.Fatalf("expected service events, got %v", set)
	}
	if _, ok := set["c"]; !ok {
		t.Fatalf("expected service event 'c' to be present")
	}
}

func TestNotifierEntryAllows(t *testing.T) {
	entry := notifierEntry{
		events: map[string]struct{}{"event-a": {}},
	}
	if !entry.allows("event-a") {
		t.Fatalf("expected event-a to be allowed")
	}
	if entry.allows("event-b") {
		t.Fatalf("did not expect event-b to be allowed")
	}

	entry.events = nil
	if !entry.allows("anything") {
		t.Fatalf("expected empty filter to allow all events")
	}
}
