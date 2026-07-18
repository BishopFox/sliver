package log

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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/natefinch/lumberjack.v2"
)

func TestNewRotatingWriterDefaults(t *testing.T) {
	w := newRotatingWriter(filepath.Join(t.TempDir(), "sliver.json"))
	if w.MaxSize != DefaultLogMaxSizeMB {
		t.Errorf("MaxSize = %d, want %d", w.MaxSize, DefaultLogMaxSizeMB)
	}
	if w.MaxBackups != DefaultLogMaxBackups {
		t.Errorf("MaxBackups = %d, want %d", w.MaxBackups, DefaultLogMaxBackups)
	}
	if w.MaxAge != DefaultLogMaxAgeDays {
		t.Errorf("MaxAge = %d, want %d", w.MaxAge, DefaultLogMaxAgeDays)
	}
	if !w.Compress {
		t.Errorf("Compress = false, want true")
	}
}

// SetRotationConfig should override on positive values and leave the built-in
// defaults untouched when passed zero / nil (backward-compatible with old
// configs that predate these fields).
func TestSetRotationConfigOverrides(t *testing.T) {
	origJSON, origTxt, origAudit := jsonRotator, txtRotator, auditRotator
	t.Cleanup(func() { jsonRotator, txtRotator, auditRotator = origJSON, origTxt, origAudit })

	jsonRotator = newRotatingWriter("json")
	txtRotator = newRotatingWriter("txt")
	auditRotator = newRotatingWriter("audit")
	all := []*lumberjack.Logger{jsonRotator, txtRotator, auditRotator}

	disable := false
	SetRotationConfig(10, 3, 7, &disable)
	for _, r := range all {
		if r.MaxSize != 10 || r.MaxBackups != 3 || r.MaxAge != 7 || r.Compress {
			t.Fatalf("override not applied: %+v", r)
		}
	}

	// Zero / nil must not clobber the values set above.
	SetRotationConfig(0, 0, 0, nil)
	for _, r := range all {
		if r.MaxSize != 10 || r.MaxBackups != 3 || r.MaxAge != 7 || r.Compress {
			t.Fatalf("zero/nil should preserve existing values: %+v", r)
		}
	}
}

// End-to-end: writing past MaxSize must rotate to a backup instead of growing
// the single file without bound (the disk-exhaustion bug this fixes).
func TestRotatingWriterRotates(t *testing.T) {
	dir := t.TempDir()
	// MaxSize is in MB (min 1). Compression off so backups are easy to count.
	w := newRotatingWriterWithLimits(filepath.Join(dir, "rotate.log"), 1, 3, 0, false)
	defer w.Close()

	chunk := make([]byte, 256*1024)
	for i := range chunk {
		chunk[i] = 'a'
	}
	for written := 0; written < 2*1024*1024; { // 2 MB total → at least one rotation
		n, err := w.Write(chunk)
		if err != nil {
			t.Fatalf("write failed: %v", err)
		}
		written += n
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	backups := 0
	for _, e := range entries {
		if e.Name() != "rotate.log" && strings.HasPrefix(e.Name(), "rotate") {
			backups++
		}
	}
	if backups == 0 {
		t.Fatalf("expected at least one rotated backup file, found none in %v", entries)
	}
}
