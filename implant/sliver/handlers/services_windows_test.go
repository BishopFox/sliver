//go:build windows

package handlers

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
	"errors"
	"testing"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// TestBuildServicesRespNilError guards against the implant-crash bug #1989:
// calling err.Error() on a nil error panics. ListServices returns nil error
// on a clean host; the response builder must handle that without panicking.
func TestBuildServicesRespNilError(t *testing.T) {
	details := []*sliverpb.ServiceDetails{
		{Name: "foo", DisplayName: "Foo Service"},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("buildServicesResp panicked on nil error: %v", r)
		}
	}()

	resp := buildServicesResp(details, nil)

	if resp == nil {
		t.Fatal("buildServicesResp returned nil")
	}
	if resp.Error != "" {
		t.Errorf("Error = %q on nil err, want empty", resp.Error)
	}
	if len(resp.Details) != 1 || resp.Details[0].Name != "foo" {
		t.Errorf("Details not preserved: %+v", resp.Details)
	}
	if resp.Response == nil {
		t.Error("Response field is nil")
	}
}

// TestBuildServicesRespWithError verifies the happy path when ListServices
// returns a non-nil error (e.g. permission denied on some services).
// Partial details must still be returned along with the error string.
func TestBuildServicesRespWithError(t *testing.T) {
	details := []*sliverpb.ServiceDetails{
		{Name: "partial"},
	}
	listErr := errors.New("spooler: access denied")

	resp := buildServicesResp(details, listErr)

	if resp.Error != "spooler: access denied" {
		t.Errorf("Error = %q, want %q", resp.Error, "spooler: access denied")
	}
	if len(resp.Details) != 1 {
		t.Errorf("Details not preserved on error path: %+v", resp.Details)
	}
}
