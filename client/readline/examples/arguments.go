package main

// This file defines a few argument choices for commands

import (
	"github.com/jessevdk/go-flags"
)

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

// loadArgumentCompletions - Adds a bunch of choices for command arguments (and their completions.)
func loadArgumentCompletions(parser *flags.Parser) {
	if parser == nil {
		return
	}
	serverCompsAddtional(parser)
}

// Additional completion mappings for command in the server context
func serverCompsAddtional(parser *flags.Parser) {

	// Stage options
	g := parser.Find("generate")
	g.FindOptionByLongName("os").Choices = implantOS
	g.FindOptionByLongName("arch").Choices = implantArch
	g.FindOptionByLongName("format").Choices = implantFmt

	// Stager options (mostly MSF)
	gs := g.Find("stager")
	gs.FindOptionByLongName("os").Choices = implantOS
	gs.FindOptionByLongName("arch").Choices = implantArch
	gs.FindOptionByLongName("protocol").Choices = msfStagerProtocols
	gs.FindOptionByLongName("msf-format").Choices = msfTransformFormats
}
