package util

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
)

func TestAllowedName(t *testing.T) {
	notAllowed := [3]string{".", "..", "..test"}
	isAllowed := [6]string{"test", "testing_string", "testing.string", "testing-string", "testing..string", "test.."}

	for i := 0; i < len(notAllowed); i++ {
		if err := AllowedName(notAllowed[i]); err == nil {
			t.Fatalf("failed to deny non allowed implant name")
		}
	}
	for i := 0; i < len(isAllowed); i++ {
		if err := AllowedName(isAllowed[i]); err != nil {
			t.Fatalf("failed to allow allowed implant name")
		}
	}

}
