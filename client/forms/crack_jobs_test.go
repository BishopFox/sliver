package forms

import (
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestCrackJobChoicesSortAndPreserveFullIDs(t *testing.T) {
	olderID := "11111111-1111-1111-1111-111111111111"
	newerID := "22222222-2222-2222-2222-222222222222"
	choices := crackJobChoices([]*clientpb.CrackJob{
		{
			ID:          olderID,
			Status:      clientpb.CrackJobStatus_COMPLETED,
			CreatedAt:   "2026-09-10T12:00:00Z",
			UpdatedAt:   100,
			Keyspace:    "500",
			ResultCount: 1,
		},
		nil,
		{},
		{
			ID:          newerID,
			Status:      clientpb.CrackJobStatus_IN_PROGRESS,
			CreatedAt:   "2026-09-11T12:00:00Z",
			UpdatedAt:   200,
			Keyspace:    "1000",
			Tasks:       []*clientpb.CrackTask{{ID: "first"}, {ID: "second"}},
			ResultCount: 3,
		},
	})
	if len(choices) != 2 {
		t.Fatalf("choices = %#v, want two valid jobs", choices)
	}
	if choices[0].id != newerID || choices[1].id != olderID {
		t.Fatalf("choice IDs = %q, %q; want newest full ID first", choices[0].id, choices[1].id)
	}
	for _, want := range []string{"JOB", "22222222", "IN_PROGRESS", "2026-09-11T12:00:00Z", "keyspace 1000", "tasks 2", "results 3"} {
		if !strings.Contains(choices[0].label, want) {
			t.Errorf("label %q does not contain %q", choices[0].label, want)
		}
	}
	if strings.Contains(choices[0].label, newerID) {
		t.Fatalf("label %q should display the compact ID", choices[0].label)
	}
}

func TestCrackJobChoicesSanitizeDisplayValues(t *testing.T) {
	choices := crackJobChoices([]*clientpb.CrackJob{{
		ID:        "12345678-1234-1234-1234-123456789abc",
		CreatedAt: "bad\nvalue\x1b[31m",
		Keyspace:  "10\t00",
	}})
	if len(choices) != 1 {
		t.Fatalf("choices = %#v", choices)
	}
	if strings.ContainsAny(choices[0].label, "\n\r\t\x1b") {
		t.Fatalf("label contains terminal control characters: %q", choices[0].label)
	}
}

func TestCrackJobSelectFormRejectsEmptyOptions(t *testing.T) {
	if _, err := CrackJobSelectForm(nil); err == nil || !strings.Contains(err.Error(), "options are required") {
		t.Fatalf("CrackJobSelectForm empty options error = %v", err)
	}
	if _, err := CrackJobSelectForm([]*clientpb.CrackJob{nil, {}}); err == nil || !strings.Contains(err.Error(), "options are required") {
		t.Fatalf("CrackJobSelectForm invalid options error = %v", err)
	}
}
