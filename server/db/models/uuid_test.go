package models

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
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	uuid "uuid"

	gosqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestUUIDCompatibility(t *testing.T) {
	const canonical = "f81d4fae-7dec-41d0-a765-00a0c91e6bf6"

	for _, value := range []string{
		canonical,
		"f81d4fae7dec41d0a76500a0c91e6bf6",
		"{" + canonical + "}",
		"{f81d4fae7dec41d0a76500a0c91e6bf6}",
		"urn:uuid:" + canonical,
		"urn:uuid:f81d4fae7dec41d0a76500a0c91e6bf6",
	} {
		parsed, err := ParseUUID(value)
		if err != nil {
			t.Fatalf("ParseUUID(%q): %v", value, err)
		}
		if got := parsed.String(); got != canonical {
			t.Fatalf("ParseUUID(%q) = %q, want %q", value, got, canonical)
		}
	}

	if _, err := ParseUUID("not-a-uuid"); err == nil {
		t.Fatal("ParseUUID accepted an invalid UUID")
	}
	if got := ParseUUIDOrNil("not-a-uuid"); got != NilUUID() {
		t.Fatalf("ParseUUIDOrNil returned %q, want nil UUID", got)
	}

	generated := NewUUID()
	standard := generated.Standard()
	if version := standard[6] >> 4; version != 4 {
		t.Fatalf("NewUUID version = %d, want 4", version)
	}
	if variant := standard[8] >> 6; variant != 2 {
		t.Fatalf("NewUUID variant = %b, want RFC 9562 variant 10", variant)
	}
	if roundTrip := UUIDFrom(standard); roundTrip != generated {
		t.Fatalf("standard UUID round trip = %q, want %q", roundTrip, generated)
	}

	raw := append([]byte(nil), standard[:]...)
	fromBytes, err := UUIDFromBytes(raw)
	if err != nil {
		t.Fatalf("UUIDFromBytes: %v", err)
	}
	if fromBytes != generated {
		t.Fatalf("UUIDFromBytes = %q, want %q", fromBytes, generated)
	}
	for _, invalidLength := range []int{0, 15, 17} {
		if _, err := UUIDFromBytes(make([]byte, invalidLength)); err == nil {
			t.Fatalf("UUIDFromBytes accepted %d bytes", invalidLength)
		}
	}
}

func TestUUIDSQLAndJSONEncoding(t *testing.T) {
	want := NewUUID()
	standard := want.Standard()

	var _ sql.Scanner = (*UUID)(nil)
	var _ driver.Valuer = UUID{}

	value, err := want.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if value != want.String() {
		t.Fatalf("Value = %#v, want %q", value, want.String())
	}

	compact := strings.ReplaceAll(want.String(), "-", "")
	for name, source := range map[string]any{
		"string":                want.String(),
		"text bytes":            []byte(want.String()),
		"raw bytes":             append([]byte(nil), standard[:]...),
		"legacy compact braces": "{" + compact + "}",
		"legacy compact urn":    "urn:uuid:" + compact,
		"model UUID":            want,
		"stdlib UUID":           standard,
	} {
		t.Run(name, func(t *testing.T) {
			var got UUID
			if err := got.Scan(source); err != nil {
				t.Fatalf("Scan: %v", err)
			}
			if got != want {
				t.Fatalf("Scan = %q, want %q", got, want)
			}
		})
	}
	if err := new(UUID).Scan(nil); err == nil {
		t.Fatal("Scan accepted nil")
	}

	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, expected := string(encoded), `"`+want.String()+`"`; got != expected {
		t.Fatalf("Marshal = %s, want %s", got, expected)
	}
	var decoded UUID
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded != want {
		t.Fatalf("Unmarshal = %q, want %q", decoded, want)
	}
}

func TestUUIDGORMRoundTrip(t *testing.T) {
	type uuidRecord struct {
		ID       UUID  `gorm:"primaryKey;type:uuid"`
		Related  UUID  `gorm:"type:uuid"`
		Optional *UUID `gorm:"type:uuid"`
	}

	databasePath := filepath.Join(t.TempDir(), "uuid.db")
	database, err := gorm.Open(gosqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDatabase, err := database.DB()
	if err != nil {
		t.Fatalf("get database handle: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDatabase.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	if err := database.AutoMigrate(&uuidRecord{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	related := NewUUID()
	optional := NewUUID()
	want := uuidRecord{ID: NewUUID(), Related: related, Optional: &optional}
	if err := database.Create(&want).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}

	var got uuidRecord
	if err := database.First(&got, "id = ?", want.ID.String()).Error; err != nil {
		t.Fatalf("load record: %v", err)
	}
	if got.ID != want.ID || got.Related != related || got.Optional == nil || *got.Optional != optional {
		t.Fatalf("database round trip = %#v, want %#v", got, want)
	}

	var storedID string
	if err := database.Raw("SELECT id FROM uuid_records WHERE id = ?", want.ID.String()).Scan(&storedID).Error; err != nil {
		t.Fatalf("read stored UUID: %v", err)
	}
	if storedID != want.ID.String() {
		t.Fatalf("stored UUID = %q, want canonical %q", storedID, want.ID.String())
	}
}

func TestUUIDMatchesStandardLibrary(t *testing.T) {
	standard := uuid.NewV4()
	model := UUIDFrom(standard)
	if model.Standard() != standard {
		t.Fatalf("Standard = %q, want %q", model.Standard(), standard)
	}
}
