package extensions

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
	"reflect"
	"testing"
)

// TestSplitExtensionArgs covers the argument-passthrough regression from #2309:
// the extension's own named flags must reach the BOF parser intact regardless of
// the cobra "--" separator, while Sliver-owned flags are still honoured.
func TestSplitExtensionArgs(t *testing.T) {
	cases := []struct {
		name    string
		in      []string
		save    bool
		timeout int64
		help    bool
		extArgs []string
	}{
		{
			name:    "named extension flags without -- separator (#2309 regression)",
			in:      []string{"--type", "6", "--fqdn", "child.htb.local"},
			timeout: defaultTimeout,
			extArgs: []string{"--type", "6", "--fqdn", "child.htb.local"},
		},
		{
			name:    "named extension flags with -- separator (existing workaround)",
			in:      []string{"--", "--type", "6", "--fqdn", "child.htb.local"},
			timeout: defaultTimeout,
			extArgs: []string{"--type", "6", "--fqdn", "child.htb.local"},
		},
		{
			name:    "sliver-owned flags precede extension arguments",
			in:      []string{"--timeout", "30", "--save", "--type", "6"},
			save:    true,
			timeout: 30,
			extArgs: []string{"--type", "6"},
		},
		{
			name:    "quoted value containing spaces stays a single argument",
			in:      []string{"--name", "hello world", "--type", "6"},
			timeout: defaultTimeout,
			extArgs: []string{"--name", "hello world", "--type", "6"},
		},
		{
			name:    "positional extension arguments are preserved",
			in:      []string{"foo", "bar", "baz"},
			timeout: defaultTimeout,
			extArgs: []string{"foo", "bar", "baz"},
		},
		{
			name:    "single-dash and double-dash forms are preserved",
			in:      []string{"-type", "6", "--fqdn", "child.htb.local"},
			timeout: defaultTimeout,
			extArgs: []string{"-type", "6", "--fqdn", "child.htb.local"},
		},
		{
			name:    "equals form for timeout",
			in:      []string{"--timeout=120", "--type", "6"},
			timeout: 120,
			extArgs: []string{"--type", "6"},
		},
		{
			name:    "short -s save flag",
			in:      []string{"-s", "--type", "6"},
			save:    true,
			timeout: defaultTimeout,
			extArgs: []string{"--type", "6"},
		},
		{
			name:    "help flag is detected",
			in:      []string{"--help"},
			help:    true,
			timeout: defaultTimeout,
			extArgs: []string{},
		},
		{
			name:    "empty input",
			in:      nil,
			timeout: defaultTimeout,
			extArgs: []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			save, timeout, help, extArgs := splitExtensionArgs(tc.in)

			if save != tc.save {
				t.Errorf("save: got %v, want %v", save, tc.save)
			}
			if timeout != tc.timeout {
				t.Errorf("timeout: got %d, want %d", timeout, tc.timeout)
			}
			if help != tc.help {
				t.Errorf("help: got %v, want %v", help, tc.help)
			}
			if !reflect.DeepEqual(extArgs, tc.extArgs) {
				t.Errorf("extArgs: got %#v, want %#v", extArgs, tc.extArgs)
			}
		})
	}
}

// TestParseBoolValue checks the --flag=value helper used for --save=value.
func TestParseBoolValue(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", true},
		{"true", true},
		{"TRUE", true},
		{"1", true},
		{"t", true},
		{"false", false},
		{"0", false},
		{"no", false}, // unsupported spellings keep the default
	}
	for _, tc := range cases {
		got := parseBoolValue(tc.in, true)
		// "no" is not a pflag spelling, so it falls through to the default (true).
		want := tc.want
		if tc.in == "no" {
			want = true
		}
		if got != want {
			t.Errorf("parseBoolValue(%q): got %v, want %v", tc.in, got, want)
		}
	}
}
