package c2

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
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// These tests exercise the pure-Go validation + translation paths in
// trigger.go. The full job-lifecycle path (UDP bind, core.Jobs.Add,
// JobCtrl watcher) is covered by the end-to-end smoke test that
// builds the actual sliver-server binary, not here.

func TestBuildKeyringRejectsEmptyConfig(t *testing.T) {
	_, err := buildKeyring(&clientpb.TriggerListenerReq{})
	if err == nil || !strings.Contains(err.Error(), "must be set") {
		t.Fatalf("expected error for empty keyring config, got %v", err)
	}
}

func TestBuildKeyringRejectsStrictWithoutClientKeys(t *testing.T) {
	_, err := buildKeyring(&clientpb.TriggerListenerReq{
		SharedSecret: []byte("default"),
		Strict:       true,
	})
	if err == nil || !strings.Contains(err.Error(), "strict") {
		t.Fatalf("expected strict-mode error, got %v", err)
	}
}

func TestBuildKeyringHappyPath(t *testing.T) {
	k, err := buildKeyring(&clientpb.TriggerListenerReq{
		SharedSecret: []byte("default-secret"),
		PerClientKeys: map[string][]byte{
			"operator-jc": []byte("jc-secret"),
		},
	})
	if err != nil {
		t.Fatalf("buildKeyring: %v", err)
	}
	if got, _ := k.SecretFor("operator-jc"); string(got) != "jc-secret" {
		t.Fatalf("per-client lookup: got %s", got)
	}
	if got, _ := k.SecretFor("anyone-else"); string(got) != "default-secret" {
		t.Fatalf("default fallback: got %s", got)
	}
}

func TestHandlerForBindingDispatchesAllKinds(t *testing.T) {
	bid := uuid.Must(uuid.NewV4()).String()
	cases := []struct {
		name    string
		binding *clientpb.TriggerIntentBinding
		want    string
	}{
		{
			"wake-beacon",
			&clientpb.TriggerIntentBinding{
				Name: "wake",
				Config: &clientpb.TriggerIntentBinding_WakeBeacon{
					WakeBeacon: &clientpb.WakeBeaconConfig{BeaconID: bid},
				},
			},
			"wake",
		},
		{
			"stop-job",
			&clientpb.TriggerIntentBinding{
				Name: "kill",
				Config: &clientpb.TriggerIntentBinding_StopJob{
					StopJob: &clientpb.StopJobConfig{JobName: "mtls-listener"},
				},
			},
			"kill",
		},
		{
			"exec",
			&clientpb.TriggerIntentBinding{
				Name: "shellout",
				Config: &clientpb.TriggerIntentBinding_Exec{
					Exec: &clientpb.ExecConfig{Cmd: "/bin/true"},
				},
			},
			"shellout",
		},
		{
			"reverse-shell",
			&clientpb.TriggerIntentBinding{
				Name: "get-shell",
				Config: &clientpb.TriggerIntentBinding_ReverseShell{
					ReverseShell: &clientpb.ReverseShellConfig{
						OperatorAddr:         "10.0.0.5:4444",
						MaxSessionDurationMs: uint32((10 * time.Minute) / time.Millisecond),
					},
				},
			},
			"get-shell",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			h, err := handlerForBinding(tc.binding)
			if err != nil {
				t.Fatalf("handlerForBinding: %v", err)
			}
			if h.Name() != tc.want {
				t.Fatalf("Name() = %q want %q", h.Name(), tc.want)
			}
		})
	}
}

func TestHandlerForBindingRejectsUnsetConfig(t *testing.T) {
	_, err := handlerForBinding(&clientpb.TriggerIntentBinding{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "config oneof") {
		t.Fatalf("expected unset-config error, got %v", err)
	}
}

func TestCoercionHelpers(t *testing.T) {
	if intOr(0, 7) != 7 {
		t.Fatalf("intOr default broken")
	}
	if intOr(42, 7) != 42 {
		t.Fatalf("intOr override broken")
	}
	if secondsOr(0, 30*time.Second) != 30*time.Second {
		t.Fatalf("secondsOr default broken")
	}
	if secondsOr(5, 30*time.Second) != 5*time.Second {
		t.Fatalf("secondsOr override broken")
	}
	if got := msOr(0, time.Second); got != time.Second {
		t.Fatalf("msOr default: %v", got)
	}
	if got := msOr(1500, time.Second); got != 1500*time.Millisecond {
		t.Fatalf("msOr override: %v", got)
	}
}

func TestSliceToSet(t *testing.T) {
	if got := sliceToSet(nil); got != nil {
		t.Fatalf("nil slice should yield nil set, got %v", got)
	}
	got := sliceToSet([]string{"a", "", "b", "a"})
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(got), got)
	}
	if _, ok := got["a"]; !ok {
		t.Fatalf("missing a")
	}
}

func TestServerIDOrDefault(t *testing.T) {
	if got := serverIDOrDefault("custom"); got != "custom" {
		t.Fatalf("expected explicit ID, got %q", got)
	}
	got := serverIDOrDefault("")
	if got == "" {
		t.Fatalf("default ID empty")
	}
}
