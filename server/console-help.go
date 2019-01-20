package main

var (
	cmdHelp = map[string]string{
		"ls":       lsHelp,
		"info":     infoHelp,
		"use":      useHelp,
		"gen":      genHelp,
		"generate": genHelp,
		"msf":      msfHelp,
		"inject":   injectHelp,
		"ps":       psHelp,
		"ping":     pingHelp,
	}

	defaultHelp = `
[[.Bold]]Commands[[.Normal]]
=========
ls     - List all sliver connections
info   - Get information about a sliver
use    - Switch the active sliver
gen    - Generate a new sliver binary
msf    - Send an msf payload to the active sliver
ps     - List processes of active sliver
ping   - Send a ping message to active sliver
inject - Inject a payload into a remote process

Use '<command> -help' to see information about a specific command.
`

	lsHelp = `
[[.Bold]]Command:[[.Normal]] ls <options>
[[.Bold]]About:[[.Normal]] List active sliver connections.
[[.Bold]]Options:[[.Normal]]
 -l | Display additional details about each sliver

`

	infoHelp = `
[[.Bold]]Command:[[.Normal]] info<sliver name>
[[.Bold]]About:[[.Normal]] Get information about a sliver by name, or for the active sliver.

`

	useHelp = `
[[.Bold]]Command:[[.Normal]] use [sliver name]
[[.Bold]]About:[[.Normal]] Switch the active sliver, a valid name must be provided (see ls).

`

	genHelp = `
[[.Bold]]Command:[[.Normal]] gen<options>
[[.Bold]]About:[[.Normal]] Generate a new sliver binary.
[[.Bold]]Options:[[.Normal]]
	       -os | [windows/linux/macos]
         -arch | [amd64/386]
       -server | Sliver server address
 -server-lport | Sliver server listen port

`
	msfHelp = `
[[.Bold]]Command:[[.Normal]] msf [-lhost] <options>
[[.Bold]]About:[[.Normal]] Execute a metasploit payload in the current process.
[[.Bold]]Options:[[.Normal]]
   -payload | The MSF payload to use (default: meterpreter_reverse_https)
   -lhost   | Metasploit listener LHOST (required)
   -lport   | Metasploit listener LPORT (default: 4444)
   -encoder | MSF encoder (default: none)
   -iter    | Iterations of the encoder (requires -encoder)

`

	injectHelp = `
[[.Bold]]Command:[[.Normal]] inject [-pid] [-lhost] <options>
[[.Bold]]About:[[.Normal]] Execute a metasploit payload in a remote process.
[[.Bold]]Options:[[.Normal]]
    -pid | The pid of the process to inject into (see 'ps')
-payload | The MSF payload to use (default: meterpreter_reverse_https)
  -lhost | Metasploit listener LHOST (required)
  -lport | Metasploit listener LPORT (default: 4444)
-encoder | MSF encoder (default: none)
   -iter | Iterations of the encoder (requires -encoder)

`

	psHelp = `
[[.Bold]]Command:[[.Normal]] ps <options>
[[.Bold]]About:[[.Normal]] List processes on remote system.
[[.Bold]]Options:[[.Normal]]
 -pid | Filter results based on pid
 -exe | Filter results based on exe name (prefix)

`

	pingHelp = `
[[.Bold]]Command:[[.Normal]] ping <sliver name>
[[.Bold]]About:[[.Normal]] Ping sliver by name or the active sliver.

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
