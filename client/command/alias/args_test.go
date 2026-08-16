package alias

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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TestSplitAliasArgs covers the argument passthrough regression from #2264:
// the alias payload's own flags must reach the payload intact, while the
// Sliver-owned flags are extracted and validated.
func TestSplitAliasArgs(t *testing.T) {
	base := aliasOwnedFlagSpecs(false)
	assembly := aliasOwnedFlagSpecs(true)

	cases := []struct {
		name    string
		specs   []ownedFlagSpec
		in      []string
		values  map[string]string
		help    bool
		extArgs []string
		wantErr bool
	}{
		{
			name:    "payload flag without -- separator (#2264 regression)",
			specs:   base,
			in:      []string{"-pid", "6969"},
			values:  map[string]string{},
			extArgs: []string{"-pid", "6969"},
		},
		{
			name:    "glued shorthand is not ours and passes through",
			specs:   base,
			in:      []string{"-pid"},
			values:  map[string]string{},
			extArgs: []string{"-pid"},
		},
		{
			name:    "payload flag with -- separator (existing workaround)",
			specs:   base,
			in:      []string{"--", "--pid", "6969"},
			values:  map[string]string{},
			extArgs: []string{"--pid", "6969"},
		},
		{
			name:    "owned flag after -- belongs to the payload",
			specs:   base,
			in:      []string{"--", "--save"},
			values:  map[string]string{},
			extArgs: []string{"--save"},
		},
		{
			name:    "long flag with separate value, then payload args",
			specs:   base,
			in:      []string{"--process", "notepad.exe", "arg1", "arg2"},
			values:  map[string]string{"process": "notepad.exe"},
			extArgs: []string{"arg1", "arg2"},
		},
		{
			name:    "long flag with equals value",
			specs:   base,
			in:      []string{"--process=notepad.exe", "arg1"},
			values:  map[string]string{"process": "notepad.exe"},
			extArgs: []string{"arg1"},
		},
		{
			name:    "shorthand with separate value",
			specs:   base,
			in:      []string{"-p", "notepad.exe", "arg1"},
			values:  map[string]string{"process": "notepad.exe"},
			extArgs: []string{"arg1"},
		},
		{
			name:    "shorthand with equals value",
			specs:   base,
			in:      []string{"-p=notepad.exe"},
			values:  map[string]string{"process": "notepad.exe"},
			extArgs: []string{},
		},
		{
			name:    "bool shorthand and long false form",
			specs:   base,
			in:      []string{"-s", "--save=false", "arg1"},
			values:  map[string]string{"save": "false"},
			extArgs: []string{"arg1"},
		},
		{
			name:    "quoted value containing spaces stays a single argument",
			specs:   base,
			in:      []string{"-A", "-kalc -kdisable", "arg1"},
			values:  map[string]string{"process-arguments": "-kalc -kdisable"},
			extArgs: []string{"arg1"},
		},
		{
			name:    "valid uint32 and int64 values",
			specs:   base,
			in:      []string{"-P", "1234", "--timeout=90"},
			values:  map[string]string{"ppid": "1234", "timeout": "90"},
			extArgs: []string{},
		},
		{
			name:    "invalid uint32 value errors",
			specs:   base,
			in:      []string{"--ppid", "notanumber"},
			wantErr: true,
		},
		{
			name:    "invalid int64 value errors",
			specs:   base,
			in:      []string{"--timeout=x"},
			wantErr: true,
		},
		{
			name:  "assembly flags are owned for assembly aliases",
			specs: assembly,
			in:    []string{"--method", "Run", "--class", "Prog", "-i", "-M", "--arch", "x64", "payloadArg"},
			values: map[string]string{
				"method": "Run", "class": "Prog", "in-process": "true",
				"amsi-bypass": "true", "arch": "x64",
			},
			extArgs: []string{"payloadArg"},
		},
		{
			name:    "assembly flags pass through for non-assembly aliases",
			specs:   base,
			in:      []string{"--method", "Run"},
			values:  map[string]string{},
			extArgs: []string{"--method", "Run"},
		},
		{
			name:    "help flags are detected",
			specs:   base,
			in:      []string{"--help"},
			help:    true,
			values:  map[string]string{},
			extArgs: []string{},
		},
		{
			name:    "short help is detected",
			specs:   base,
			in:      []string{"-h"},
			help:    true,
			values:  map[string]string{},
			extArgs: []string{},
		},
		{
			name:    "value flag as last token keeps its default",
			specs:   base,
			in:      []string{"arg1", "--process"},
			values:  map[string]string{},
			extArgs: []string{"arg1"},
		},
		{
			name:    "empty input",
			specs:   base,
			in:      nil,
			values:  map[string]string{},
			extArgs: []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values, help, extArgs, err := splitAliasArgs(tc.in, tc.specs)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got values=%#v help=%v extArgs=%#v", values, help, extArgs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if help != tc.help {
				t.Errorf("help: got %v, want %v", help, tc.help)
			}
			if !reflect.DeepEqual(values, tc.values) {
				t.Errorf("values: got %#v, want %#v", values, tc.values)
			}
			if !reflect.DeepEqual(extArgs, tc.extArgs) {
				t.Errorf("extArgs: got %#v, want %#v", extArgs, tc.extArgs)
			}
		})
	}
}

// TestApplyOwnedAliasFlags checks that manually parsed values are re-published
// on the command's flag set for every value type an alias can own.
func TestApplyOwnedAliasFlags(t *testing.T) {
	cmd := &cobra.Command{}
	f := pflag.NewFlagSet("alias-test", pflag.ContinueOnError)
	f.StringP("process", "p", "", "host process")
	f.StringP("process-arguments", "A", "", "host process arguments")
	f.Uint32P("ppid", "P", 0, "parent pid")
	f.BoolP("save", "s", false, "save output")
	f.Int64P("timeout", "t", 60, "timeout")
	f.BoolP("amsi-bypass", "M", false, "amsi bypass")
	cmd.Flags().AddFlagSet(f)

	err := applyOwnedAliasFlags(cmd, map[string]string{
		"process":           "notepad.exe",
		"process-arguments": "-k foo",
		"ppid":              "1234",
		"save":              "true",
		"timeout":           "90",
		"amsi-bypass":       "true",
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	for name, want := range map[string]interface{}{
		"process": "notepad.exe", "process-arguments": "-k foo",
	} {
		got, _ := cmd.Flags().GetString(name)
		if got != want.(string) {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}
	if ppid, _ := cmd.Flags().GetUint32("ppid"); ppid != 1234 {
		t.Errorf("ppid: got %d, want 1234", ppid)
	}
	if save, _ := cmd.Flags().GetBool("save"); !save {
		t.Error("save: got false, want true")
	}
	if timeout, _ := cmd.Flags().GetInt64("timeout"); timeout != 90 {
		t.Errorf("timeout: got %d, want 90", timeout)
	}
	if amsi, _ := cmd.Flags().GetBool("amsi-bypass"); !amsi {
		t.Error("amsi-bypass: got false, want true")
	}

	// Unset flags keep their defaults.
	if inProc := cmd.Flags().Lookup("in-process"); inProc != nil {
		t.Error("in-process should not be registered on this command")
	}
}
