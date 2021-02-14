package completers

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
	// Logs & components
	logLevels = []string{"trace", "debug", "info", "warning", "error"}
	loggers   = []string{"client", "comm"}

	// Stages / Stagers
	implantOS   = []string{"windows", "linux", "darwin"}
	implantArch = []string{"amd64", "x86"}
	implantFmt  = []string{"exe", "shared", "service", "shellcode"}

	stageListenerProtocols = []string{"tcp", "http", "https"}

	// MSF
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

	msfPayloads = map[string][]string{
		"windows": windowsMsfPayloads,
		"linux":   linuxMsfPayloads,
		"osx":     osxMsfPayloads,
	}

	// ValidPayloads - Valid payloads and OS combos
	windowsMsfPayloads = []string{
		"meterpreter_reverse_http",
		"meterpreter_reverse_https",
		"meterpreter_reverse_tcp",
		"meterpreter/reverse_tcp",
		"meterpreter/reverse_http",
		"meterpreter/reverse_https",
	}
	linuxMsfPayloads = []string{
		"meterpreter_reverse_http",
		"meterpreter_reverse_https",
		"meterpreter_reverse_tcp",
	}
	osxMsfPayloads = []string{
		"meterpreter_reverse_http",
		"meterpreter_reverse_https",
		"meterpreter_reverse_tcp",
	}

	// Comm network protocols
	portfwdProtocols     = []string{"tcp", "udp"}
	transportProtocols   = []string{"tcp", "udp", "ip"}
	applicationProtocols = []string{"http", "https", "mtls", "quic", "http3", "dns", "named_pipe"}
)
