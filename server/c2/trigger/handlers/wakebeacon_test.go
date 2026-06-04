package handlers

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/0x90pkt/trigger/pkg/intents"
	"github.com/gofrs/uuid"

	"github.com/bishopfox/sliver/server/db/models"
)

// withBeaconStubs swaps the package-level indirections for the
// duration of the test, restoring originals on cleanup. Each test that
// touches the DB uses this — keeps tests hermetic.
func withBeaconStubs(t *testing.T, lookup func(string) (*models.Beacon, error), save func(*models.Beacon) error) {
	t.Helper()
	origLookup := beaconLookup
	origSave := beaconSave
	beaconLookup = lookup
	beaconSave = save
	t.Cleanup(func() {
		beaconLookup = origLookup
		beaconSave = origSave
	})
}

func TestNewWakeSessionRejectsBadInputs(t *testing.T) {
	valid := uuid.Must(uuid.NewV4()).String()
	cases := []struct {
		name     string
		intent   string
		beaconID string
		want     string
	}{
		{"empty task name", "", valid, "task name"},
		{"empty beacon id", "wake", "", "beacon ID"},
		{"non-uuid beacon id", "wake", "not-a-uuid", "invalid UUID"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewWakeSession(tc.intent, tc.beaconID)
			if err == nil {
				t.Fatalf("expected NewWakeSession to reject %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q did not contain %q", err.Error(), tc.want)
			}
		})
	}
}

func TestNewWakeBeaconAliasWorks(t *testing.T) {
	id := uuid.Must(uuid.NewV4()).String()
	h, err := NewWakeBeacon("alias-test", id)
	if err != nil {
		t.Fatalf("NewWakeBeacon (alias): %v", err)
	}
	if h.Name() != "alias-test" {
		t.Fatalf("Name() = %q, want alias-test", h.Name())
	}
}

func TestWakeSessionExecuteUpdatesNextCheckin(t *testing.T) {
	beaconID := uuid.Must(uuid.NewV4())
	b := &models.Beacon{ID: beaconID, NextCheckin: 1}

	var savedBeacon *models.Beacon
	withBeaconStubs(t,
		func(id string) (*models.Beacon, error) {
			if id != beaconID.String() {
				return nil, errors.New("wrong id")
			}
			return b, nil
		},
		func(beacon *models.Beacon) error {
			savedBeacon = beacon
			return nil
		},
	)

	h, err := NewWakeSession("wake-jumpbox", beaconID.String())
	if err != nil {
		t.Fatalf("NewWakeSession: %v", err)
	}

	before := time.Now().Unix()
	if err := h.Execute(context.Background(), intents.Event{ClientID: "operator-jc", SourceIP: "10.0.0.5"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if savedBeacon == nil {
		t.Fatalf("save was not called")
	}
	if savedBeacon.NextCheckin < before {
		t.Fatalf("NextCheckin=%d not updated (before=%d)", savedBeacon.NextCheckin, before)
	}
}

func TestWakeSessionExecuteHandlesSessionMode(t *testing.T) {
	// When beacon lookup returns nil (session-mode trigger implant),
	// Execute should succeed without calling save.
	beaconID := uuid.Must(uuid.NewV4())
	withBeaconStubs(t,
		func(_ string) (*models.Beacon, error) { return nil, nil },
		func(_ *models.Beacon) error { t.Fatal("save should not be called for session-mode wake"); return nil },
	)
	h, _ := NewWakeSession("wake-session", beaconID.String())
	err := h.Execute(context.Background(), intents.Event{ClientID: "operator-jc", SourceIP: "10.0.0.5"})
	if err != nil {
		t.Fatalf("expected no error for session-mode wake, got: %v", err)
	}
}

func TestWakeSessionExecuteHandlesBeaconNotFoundGracefully(t *testing.T) {
	// With session-mode trigger implants, beacon not found is expected
	// and should NOT be an error.
	beaconID := uuid.Must(uuid.NewV4())
	withBeaconStubs(t,
		func(_ string) (*models.Beacon, error) { return nil, nil },
		func(_ *models.Beacon) error { t.Fatal("save should not be called"); return nil },
	)
	h, _ := NewWakeSession("wake", beaconID.String())
	err := h.Execute(context.Background(), intents.Event{})
	if err != nil {
		t.Fatalf("expected no error for session-mode (beacon not found), got: %v", err)
	}
}

func TestWakeSessionExecutePropagatesLookupError(t *testing.T) {
	beaconID := uuid.Must(uuid.NewV4())
	withBeaconStubs(t,
		func(_ string) (*models.Beacon, error) { return nil, errors.New("db down") },
		func(_ *models.Beacon) error { t.Fatal("save should not be called"); return nil },
	)
	h, _ := NewWakeSession("wake", beaconID.String())
	err := h.Execute(context.Background(), intents.Event{})
	if err == nil || !strings.Contains(err.Error(), "db down") {
		t.Fatalf("expected db error to propagate, got %v", err)
	}
}

func TestWakeSessionExecutePropagatesSaveError(t *testing.T) {
	beaconID := uuid.Must(uuid.NewV4())
	withBeaconStubs(t,
		func(_ string) (*models.Beacon, error) { return &models.Beacon{ID: beaconID}, nil },
		func(_ *models.Beacon) error { return errors.New("save failed") },
	)
	h, _ := NewWakeSession("wake", beaconID.String())
	err := h.Execute(context.Background(), intents.Event{})
	if err == nil || !strings.Contains(err.Error(), "save failed") {
		t.Fatalf("expected save error to propagate, got %v", err)
	}
}

func TestWakeSessionName(t *testing.T) {
	h, _ := NewWakeSession("custom-task", uuid.Must(uuid.NewV4()).String())
	if h.Name() != "custom-task" {
		t.Fatalf("Name() = %q, want custom-task", h.Name())
	}
}
