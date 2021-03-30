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

	// Stages
	implantFmt = []string{"exe", "shared", "service", "shellcode"}

	// Reference:
	// https://github.com/golang/go/blob/master/src/go/build/syslist.go
	osArchs = []string{
		"aix/ppc64",
		"darwin/amd64",
		"darwin/arm64",
		"dragonfly/amd64",
		"freebsd/386",
		"freebsd/amd64",
		"freebsd/arm",
		"freebsd/arm64",
		"illumos/amd64",
		"js/wasm",
		"linux/386",
		"linux/amd64",
		"linux/arm",
		"linux/arm64",
		"linux/ppc64",
		"linux/ppc64le",
		"linux/mips",
		"linux/mipsle",
		"linux/mips64",
		"linux/mips64le",
		"linux/riscv64",
		"linux/s390x",
		"netbsd/386",
		"netbsd/amd64",
		"netbsd/arm",
		"netbsd/arm64",
		"openbsd/386",
		"openbsd/amd64",
		"openbsd/arm",
		"openbsd/arm64",
		"plan9/386",
		"plan9/amd64",
		"plan9/arm",
		"solaris/amd64",
		"windows/386",
		"windows/amd64",
		"windows/arm",
	}

	// All of these architectures work with a simple Go compiler, no need for external ones.
	// validArchsGoNative = map[string][]string{
	//         "aix":       []string{"ppc64"},
	//         "android":   []string{"x86", "amd64", "arm", "arm64"},
	//         "darwin":    []string{"amd64", "arm64"},
	//         "dragonfly": []string{"amd64", "arm64"},
	//         "freebsd":   []string{"x86", "amd64", "arm", "arm64"},
	//         "illumos":   []string{"amd64"},
	//         "ios":       []string{"amd64", "arm64"},
	//         "js":        []string{"wasm"},
	//         "linux":     []string{"x86", "amd64", "arm", "arm64", "mips", "mips64", "mips64le", "mipsle", "ppc64", "ppc64le", "riscv64", "s390x"},
	//         "netbsd":    []string{"x86", "amd64", "arm", "arm64"},
	//         "openbsd":   []string{"x86", "amd64", "arm", "arm64", "mips64"},
	//         "plan9":     []string{"x86", "amd64", "arm"},
	//         "solaris":   []string{"amd64"},
	//         "windows":   []string{"x86", "amd64", "arm"},
	// }

	// Stagers
	stageListenerProtocols = []string{"tcp", "http", "https"}

	// Assembly architectures
	assemblyArchs = []string{"x86", "x64", "x84"}

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
