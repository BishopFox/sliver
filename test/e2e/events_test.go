package e2e

import (
	"context"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestEventHubRejectsCursorOlderThanRetainedHistory(t *testing.T) {
	hub := &eventHub{
		base:    17,
		history: []*clientpb.Event{{EventType: "retained"}},
		wake:    make(chan struct{}, 1),
		done:    make(chan struct{}),
	}

	event, next, err := hub.wait(context.Background(), 16, func(*clientpb.Event) bool { return true })
	if err == nil {
		t.Fatal("expected stale cursor error")
	}
	if event != nil {
		t.Fatalf("event got %#v, want nil", event)
	}
	if next != hub.base {
		t.Fatalf("next cursor got %d, want retained base %d", next, hub.base)
	}
	for _, text := range []string{"history overflow", "cursor 16", "base 17"} {
		if !strings.Contains(err.Error(), text) {
			t.Fatalf("error %q missing %q", err, text)
		}
	}
}

func TestEventHubReturnsMatchingRetainedEvent(t *testing.T) {
	hub := &eventHub{
		base: 5,
		history: []*clientpb.Event{
			{EventType: "ignore"},
			{EventType: "match"},
		},
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
	}

	event, next, err := hub.wait(context.Background(), 5, func(event *clientpb.Event) bool {
		return event.EventType == "match"
	})
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if event == nil || event.EventType != "match" {
		t.Fatalf("event got %#v", event)
	}
	if next != 7 {
		t.Fatalf("next cursor got %d, want 7", next)
	}
}
