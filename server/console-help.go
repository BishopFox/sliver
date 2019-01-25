package main

var (
	cmdHelp = map[string]string{
		sessionsStr:   sessionsHelp,
		backgroundStr: backgroundHelp,
		infoStr:       infoHelp,
		useStr:        useHelp,
		generateStr:   genHelp,
		msfStr:        msfHelp,
		injectStr:     injectHelp,
		psStr:         psHelp,
		pingStr:       pingHelp,
		killStr:       killHelp,
		lsStr:         lsHelp,
		cdStr:         cdHelp,
		catStr:        catHelp,
		downloadStr:   downloadHelp,
		uploadStr:     uploadHelp,
	}

	defaultHelp = `
[[.Bold]]Commands[[.Normal]]
=========
sessions - List all sliver connections
info     - Get information about a sliver
use      - Switch the active sliver
generate - Generate a new sliver binary
msf      - Send an msf payload to the active sliver
ps       - List processes of active sliver
ping     - Send a ping message to active sliver
inject   - Inject a payload into a remote process
kill     - Kill a remote sliver process

Use '<command> -help' to see information about a specific command.
`

	sessionsHelp = `
[[.Bold]]Command:[[.Normal]] sessions <options>
[[.Bold]]About:[[.Normal]] List files on remote system.
[[.Bold]]Options:[[.Normal]]
	-i | Interact with sliver
`

	backgroundHelp = `
[[.Bold]]Command:[[.Normal]] background
[[.Bold]]About:[[.Normal]] Background the active sliver.

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
 -lhost | Sliver server address
 -lport | Sliver server listen port

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

	killHelp = `
[[.Bold]]Command:[[.Normal]] kill <sliver name>
[[.Bold]]About:[[.Normal]] Kill a remote sliver process (does not delete file).

`

	lsHelp = `
[[.Bold]]Command:[[.Normal]] ls
[[.Bold]]About:[[.Normal]] List remote files in current directory.

`

	cdHelp = `
[[.Bold]]Command:[[.Normal]] cd
[[.Bold]]About:[[.Normal]] Change working directory.

`

	pwdHelp = `
[[.Bold]]Command:[[.Normal]] pwd
[[.Bold]]About:[[.Normal]] Print working directory.

`

	catHelp = `
[[.Bold]]Command:[[.Normal]] cat <remote file> 
[[.Bold]]About:[[.Normal]] Cat a remote file to stdout.

`

	downloadHelp = `
[[.Bold]]Command:[[.Normal]] download <remote src> <local dest>
[[.Bold]]About:[[.Normal]] Download a file from the remote system.

`

	uploadHelp = `
[[.Bold]]Command:[[.Normal]] upload <local src> <remote dest>
[[.Bold]]About:[[.Normal]] Upload a file to the remote system.

`
)

func getHelpFor(cmdName string) string {
	if 0 < len(cmdName) {
		if help, ok := cmdHelp[cmdName]; ok {
			return help
		}
	}
	return defaultHelp
}
