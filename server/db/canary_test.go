//go:build go_sqlite

package db

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
	"testing"

	"github.com/bishopfox/sliver/server/db/models"
	gosqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initTestDB replaces the global Client with an in-memory SQLite instance
// and migrates only the models needed for canary tests.
func initTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(gosqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	if err := db.AutoMigrate(&models.DNSCanary{}); err != nil {
		t.Fatalf("migrate DNSCanary: %v", err)
	}
	Client = db
}

// TestCanaryByDomainNotFound verifies that CanaryByDomain returns nil (not a
// zero-value protobuf) when no record exists.  The original code returned
// canary.ToProtobuf() unconditionally, so callers could not distinguish
// "not found" from "found" via a nil check.
func TestCanaryByDomainNotFound(t *testing.T) {
	initTestDB(t)

	got, err := CanaryByDomain("does-not-exist.example.com")
	if err == nil {
		t.Fatal("expected error for missing domain, got nil")
	}
	if got != nil {
		t.Errorf("CanaryByDomain (not found) returned non-nil protobuf: %+v", got)
	}
}

// TestCanaryByDomainFound verifies that CanaryByDomain returns a populated
// protobuf and nil error when the domain exists.
func TestCanaryByDomainFound(t *testing.T) {
	initTestDB(t)

	// Seed a canary record directly via GORM model.
	seed := &models.DNSCanary{
		ImplantName: "TEST_IMPLANT",
		Domain:      "abc123.canary.example.com.",
		Triggered:   false,
		Count:       0,
	}
	if err := Session().Create(seed).Error; err != nil {
		t.Fatalf("seed canary: %v", err)
	}

	got, err := CanaryByDomain("abc123.canary.example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("CanaryByDomain returned nil for existing domain")
	}
	if got.ImplantName != "TEST_IMPLANT" {
		t.Errorf("ImplantName = %q, want %q", got.ImplantName, "TEST_IMPLANT")
	}
	if got.Domain != "abc123.canary.example.com." {
		t.Errorf("Domain = %q, want %q", got.Domain, "abc123.canary.example.com.")
	}
}
