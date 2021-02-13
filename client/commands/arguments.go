package commands

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/jessevdk/go-flags"
)

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

// Command/option argument choices
var (
	implantOS   = []string{"windows", "linux", "darwin"}
	implantArch = []string{"amd64", "x86"}
	implantFmt  = []string{"exe", "shared", "service", "shellcode"}

	stageListenerProtocols = []string{"tcp", "http", "https"}

	msfStagerProtocols  = []string{"tcp", "http", "https"}
	msfTransformFormats = []string{
		"bash",
		"c",
		"csharp",
		"dw",
		"dword",
		"hex",
		"java",
		"js_be",
		"js_le",
		"num",
		"perl",
		"pl",
		"powershell",
		"ps1",
		"py",
		"python",
		"raw",
		"rb",
		"ruby",
		"sh",
		"vbapplication",
		"vbscript",
	}

	msfEncoders = []string{
		"x86/shikata_ga_nai",
		"x64/xor_dynamic",
	}

	msfPayloadNames = []string{
		"meterpreter_reverse_tcp",
		"meterpreter_reverse_http",
		"meterpreter_reverse_https",
		"meterpreter_bind_tcp",
	}

	// ValidPayloads - Valid payloads and OS combos
	validPayloads = map[string]map[string]bool{
		"windows": {
			"meterpreter_reverse_http":  true,
			"meterpreter_reverse_https": true,
			"meterpreter_reverse_tcp":   true,
			"meterpreter/reverse_tcp":   true,
			"meterpreter/reverse_http":  true,
			"meterpreter/reverse_https": true,
		},
		"linux": {
			"meterpreter_reverse_http":  true,
			"meterpreter_reverse_https": true,
			"meterpreter_reverse_tcp":   true,
		},
		"osx": {
			"meterpreter_reverse_http":  true,
			"meterpreter_reverse_https": true,
			"meterpreter_reverse_tcp":   true,
		},
	}

	transportProtocols   = []string{"tcp", "udp", "ip"}
	applicationProtocols = []string{"http", "https", "mtls", "quic", "http3", "dns", "named_pipe"}
)

// ArgumentByName Get the name of a detected command's argument
func ArgumentByName(command *flags.Command, name string) *flags.Arg {
	args := command.Args()
	for _, arg := range args {
		if arg.Name == name {
			return arg
		}
	}

	// Maybe we can check for aliases, later...
	// Might sometimes push interesting things...

	return nil
}

// OptionByName - Returns an option for a command or a subcommand, identified by name
func OptionByName(cmd *flags.Command, option string) *flags.Option {

	if cmd == nil {
		return nil
	}
	// Get all (root) option groups.
	groups := cmd.Groups()

	// For each group, build completions
	for _, grp := range groups {
		// Add each option to completion group
		for _, opt := range grp.Options() {
			if opt.LongName == option {
				return opt
			}
		}
	}
	return nil
}

// GetSession - Get session by session ID or name
func GetSession(arg string) *clientpb.Session {
	sessions, err := transport.RPC.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return nil
	}
	for _, session := range sessions.GetSessions() {
		if fmt.Sprintf("%d", session.ID) == arg {
			return session
		}
	}
	return nil
}
