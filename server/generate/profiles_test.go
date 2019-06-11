package generate

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

func TestProfileByName(t *testing.T) {
	name := "foobar"
	config := &SliverConfig{
		GOOS:   "windows",
		GOARCH: "amd64",
	}
	err := ProfileSave(name, config)
	if err != nil {
		t.Errorf("%v", err)
	}

	profile, err := ProfileByName(name)
	if err != nil {
		t.Errorf("%v", err)
	}

	if profile.GOOS != config.GOOS {
		t.Errorf("Fetched data does not match saved data")
	}
}
