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
	"strings"
	"testing"
	"time"

	"github.com/0x90pkt/trigger/pkg/intents"

	"github.com/bishopfox/sliver/server/core"
)

// withJobStubs swaps the jobsLookup indirection for the test duration.
func withJobStubs(t *testing.T, lookup func() []*core.Job) {
	t.Helper()
	orig := jobsLookup
	jobsLookup = lookup
	t.Cleanup(func() { jobsLookup = orig })
}

func TestNewStopJobRejectsBadInputs(t *testing.T) {
	if _, err := NewStopJob("", "x"); err == nil {
		t.Fatalf("expected error for empty task name")
	}
	if _, err := NewStopJob("x", ""); err == nil {
		t.Fatalf("expected error for empty job name")
	}
}

func TestStopJobExecuteSendsOnJobCtrl(t *testing.T) {
	job := &core.Job{ID: 7, Name: "mtls-8443", JobCtrl: make(chan bool, 1)}
	withJobStubs(t, func() []*core.Job { return []*core.Job{job} })

	h, err := NewStopJob("kill-mtls", "mtls-8443")
	if err != nil {
		t.Fatalf("NewStopJob: %v", err)
	}
	if err := h.Execute(context.Background(), intents.Event{ClientID: "operator-jc"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	select {
	case got := <-job.JobCtrl:
		if !got {
			t.Fatalf("JobCtrl received %v, want true", got)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected value on JobCtrl, got timeout")
	}
}

func TestStopJobExecuteHandlesNoMatchingJob(t *testing.T) {
	withJobStubs(t, func() []*core.Job {
		return []*core.Job{{ID: 1, Name: "different-job", JobCtrl: make(chan bool, 1)}}
	})
	h, _ := NewStopJob("kill", "missing")
	err := h.Execute(context.Background(), intents.Event{})
	if err == nil || !strings.Contains(err.Error(), "no active job") {
		t.Fatalf("expected no-active-job error, got %v", err)
	}
}

func TestStopJobExecuteNonBlockingWhenCtrlBusy(t *testing.T) {
	// Unbuffered channel, no reader → send must NOT block; handler
	// returns "busy" error instead of hanging the worker goroutine.
	job := &core.Job{ID: 9, Name: "stuck", JobCtrl: make(chan bool)}
	withJobStubs(t, func() []*core.Job { return []*core.Job{job} })

	h, _ := NewStopJob("kill-stuck", "stuck")
	done := make(chan error, 1)
	go func() { done <- h.Execute(context.Background(), intents.Event{}) }()

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "busy") {
			t.Fatalf("expected busy error, got %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Execute hung — non-blocking send is broken")
	}
}

func TestStopJobExecutePicksFirstMatch(t *testing.T) {
	first := &core.Job{ID: 1, Name: "dupe", JobCtrl: make(chan bool, 1)}
	second := &core.Job{ID: 2, Name: "dupe", JobCtrl: make(chan bool, 1)}
	withJobStubs(t, func() []*core.Job { return []*core.Job{first, second} })

	h, _ := NewStopJob("kill", "dupe")
	if err := h.Execute(context.Background(), intents.Event{}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Exactly one of the two should have received the signal — the
	// first match by iteration order.
	select {
	case <-first.JobCtrl:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("first job did not receive signal")
	}
	select {
	case <-second.JobCtrl:
		t.Fatalf("second job should NOT have received signal")
	case <-time.After(50 * time.Millisecond):
		// expected: nothing received
	}
}

func TestStopJobName(t *testing.T) {
	h, _ := NewStopJob("custom-stop", "target")
	if h.Name() != "custom-stop" {
		t.Fatalf("Name() = %q, want custom-stop", h.Name())
	}
}
