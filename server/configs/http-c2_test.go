package configs

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"encoding/json"
	"testing"
)

var (
	fileExtCoercions = map[string]string{
		"js":    "js",
		".js":   "js",
		"/.js":  "js",
		"/.es6": "es6",
		".mp4":  "mp4",
	}
)

func TestCoerceFileExt(t *testing.T) {
	for input, output := range fileExtCoercions {
		if value := coerceFileExt(input); value != output {
			t.Fatalf("'%s' was parsed as '%s', expected '%s'", input, value, output)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	data, err := json.Marshal(defaultHTTPC2Config)
	if err != nil {
		t.Fatal(err)
	}
	var config *HTTPC2Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		t.Fatal(err)
	}
	err = checkHTTPC2Config(config)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPollConfig(t *testing.T) {

	// Missing PollFileExt
	config := defaultHTTPC2Config
	origPollFileExt := config.ImplantConfig.PollFileExt
	for _, ext := range []string{"", ".", "..."} {
		config.ImplantConfig.PollFileExt = ext
		err := checkHTTPC2Config(&config)
		if err != ErrMissingPollFileExt {
			t.Fatalf("Parsed '%s' as not missing (%s)", ext, config.ImplantConfig.PollFileExt)
		}
	}
	config.ImplantConfig.PollFileExt = origPollFileExt

	// Missing PollFiles
	emptyPollFiles := [][]string{
		{},
		{""},
		{"/"},
		{"", "", ""},
		{"/", "/", "/"},
	}
	origPollFiles := config.ImplantConfig.PollFiles
	for _, empty := range emptyPollFiles {
		config.ImplantConfig.PollFiles = empty
		err := checkHTTPC2Config(&config)
		if err != ErrTooFewPollFiles {
			t.Fatalf("Expected too few poll files from %v got %v", empty, err)
		}
	}
	config.ImplantConfig.PollFiles = origPollFiles

}
