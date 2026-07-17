package burn

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
	"os"
	"path/filepath"
	"testing"
	"time"
)

// All tests use SkipSelf + NoExit so the test binary stays alive.

func TestNowWipesExtraPaths(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		filepath.Join(dir, "a.log"),
		filepath.Join(dir, "b.cache"),
		filepath.Join(dir, "nested", "c.tmp"),
	}
	_ = os.MkdirAll(filepath.Join(dir, "nested"), 0o755)
	for _, f := range files {
		if err := os.WriteFile(f, []byte("forensic-trace"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	Now(Options{
		Reason:     ReasonManual,
		ExtraPaths: files,
		SkipSelf:   true,
		NoExit:     true,
	})

	for _, f := range files {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Fatalf("path %q still exists after burn: %v", f, err)
		}
	}
}

func TestNowWipesPersistencePaths(t *testing.T) {
	dir := t.TempDir()
	unit := filepath.Join(dir, "fake.service")
	if err := os.WriteFile(unit, []byte("[Unit]\nDescription=fake\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	Now(Options{
		Persistence: []string{unit},
		SkipSelf:    true,
		NoExit:      true,
	})

	if _, err := os.Stat(unit); !os.IsNotExist(err) {
		t.Fatalf("persistence file still exists after burn")
	}
}

func TestNowEmitsAuditFirst(t *testing.T) {
	got := struct {
		called bool
		reason Reason
		when   time.Time
	}{}
	SetAuditEmitter(&fakeEmitter{
		onEmit: func(r Reason, w time.Time, _ Options) {
			got.called = true
			got.reason = r
			got.when = w
		},
	})
	t.Cleanup(func() { SetAuditEmitter(nil) })

	Now(Options{
		Reason:   ReasonTTLExpired,
		SkipSelf: true,
		NoExit:   true,
	})

	if !got.called {
		t.Fatalf("emitter was not called")
	}
	if got.reason != ReasonTTLExpired {
		t.Fatalf("reason: got %q want %q", got.reason, ReasonTTLExpired)
	}
}

func TestNowSurvivesPanickingEmitter(t *testing.T) {
	SetAuditEmitter(&fakeEmitter{
		onEmit: func(Reason, time.Time, Options) {
			panic("emitter blew up")
		},
	})
	t.Cleanup(func() { SetAuditEmitter(nil) })

	// Should NOT propagate the panic — burn must always run.
	dir := t.TempDir()
	f := filepath.Join(dir, "must-be-wiped")
	_ = os.WriteFile(f, []byte("x"), 0o644)

	Now(Options{
		ExtraPaths: []string{f},
		SkipSelf:   true,
		NoExit:     true,
	})

	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Fatalf("burn aborted on emitter panic — wipe never happened")
	}
}

func TestNowDefaultReasonIsManual(t *testing.T) {
	captured := struct{ reason Reason }{}
	SetAuditEmitter(&fakeEmitter{
		onEmit: func(r Reason, _ time.Time, _ Options) { captured.reason = r },
	})
	t.Cleanup(func() { SetAuditEmitter(nil) })

	Now(Options{SkipSelf: true, NoExit: true})

	if captured.reason != ReasonManual {
		t.Fatalf("default reason: got %q want %q", captured.reason, ReasonManual)
	}
}

type fakeEmitter struct {
	onEmit func(Reason, time.Time, Options)
}

func (e *fakeEmitter) EmitBurn(reason Reason, when time.Time, opts Options) {
	if e.onEmit != nil {
		e.onEmit(reason, when, opts)
	}
}
