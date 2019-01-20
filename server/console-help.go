package main

var (
	cmdHelp = map[string]string{
		"ls":       lsHelp,
		"info":     infoHelp,
		"use":      useHelp,
		"gen":      genHelp,
		"generate": genHelp,
		"msf":      msfHelp,
	}

	defaultHelp = `
[[.Bold]]Commands[[.Normal]]
=========
ls   - List all sliver connections
info - Get information about a sliver
use  - Switch the active sliver
gen  - Generate a new sliver binary
msf  - Send an msf payload to the active sliver
kill - Kill connection to sliver

Use 'help <command>' to see information about a specific command.
`

	lsHelp = `
Command: [[.Bold]]ls[[.Normal]]
List active sliver connections.

`

	infoHelp = `
Command: [[.Bold]]info[[.Normal]] <sliver name>
Get information about a sliver by name, or for the active sliver.

`

	useHelp = `
Command: [[.Bold]]use[[.Normal]] [sliver name]
Switch the active sliver, a valid name must be provided (see ls).

`

	genHelp = `
Command: [[.Bold]]gen[[.Normal]] <options>
Options:
	      -os  | [windows/linux/macos]
        -arch  | [amd64/386]
       -server | Sliver server address
 -server-lport | Sliver server listen port

`
	msfHelp = `
Command: [[.Bold]]msf[[.Normal]] <options>
Options:
   -payload | The MSF payload to use (default: meterpreter_reverse_https)
   -lhost   | Metasploit listener LHOST (required)
   -lport   | Metasploit listener LPORT (default: 4444)
   -encoder | MSF encoder
   -iter    | Iterations of the encoder

`
)

func getHelpFor(args []string) string {
	if 0 < len(args) {
		if help, ok := cmdHelp[args[0]]; ok {
			return help
		}
	}
	return defaultHelp
}
