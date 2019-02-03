package console

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
		mkdirStr:      mkdirHelp,
		rmStr:         rmHelp,
		procdumpStr:   procdumpHelp,
	}

	defaultHelp = `
[[.Bold]]Commands[[.Normal]]
========
  sessions - List sliver connections
background - Background the active sliver
      info - Get information about a sliver
       use - Select an active sliver
  generate - Generate a new sliver binary
        ps - List processes of active sliver
      ping - Send a ping message to active sliver
	  kill - Kill a remote sliver process

[[.Bold]]File System Commands[[.Normal]]
====================
      ls - List remote directory
     pwd - Print the current remote working directory
   mkdir - Make a directory on the remote file system
      rm - Delete a remote file system path
     cat - Dump a remote file to local stdout
download - Download a remote file to the local system
  upload - Upload a local file to the remote system

[[.Bold]]Chainloader Commands[[.Normal]]
====================
   msf - Send an msf payload to the active sliver
inject - Inject a payload into a remote process of the active sliver


Use 'help <command>' to see information about a specific command.

`

	sessionsHelp = `
[[.Bold]]Command:[[.Normal]] sessions <options>
[[.Bold]]About:[[.Normal]] List files on remote system.
[[.Bold]]Options:[[.Normal]]
--interact | Interact with sliver (same as 'use')

`

	backgroundHelp = `
[[.Bold]]Command:[[.Normal]] background
[[.Bold]]About:[[.Normal]] Background the active sliver.

`

	infoHelp = `
[[.Bold]]Command:[[.Normal]] info <sliver name>
[[.Bold]]About:[[.Normal]] Get information about a sliver by name, or for the active sliver.

`

	useHelp = `
[[.Bold]]Command:[[.Normal]] use [sliver name]
[[.Bold]]About:[[.Normal]] Switch the active sliver, a valid name must be provided (see sessions).

`

	genHelp = `
[[.Bold]]Command:[[.Normal]] gen <options>
[[.Bold]]About:[[.Normal]] Generate a new sliver binary.
[[.Bold]]Options:[[.Normal]]
    --os | [windows/linux/macos] (default: windows)
  --arch | [amd64/386] (default: amd64)
 --lhost | Sliver server address (required)
 --lport | Sliver server listen port (default: 8888)

`
	msfHelp = `
[[.Bold]]Command:[[.Normal]] msf [--lhost] <options>
[[.Bold]]About:[[.Normal]] Execute a metasploit payload in the current process.
[[.Bold]]Options:[[.Normal]]
   --payload | The MSF payload to use (default: meterpreter_reverse_https)
     --lhost | Metasploit listener LHOST (required)
     --lport | Metasploit listener LPORT (default: 4444)
   --encoder | MSF encoder (default: none)
--iterations | Iterations of the encoder (requires -encoder)

`

	injectHelp = `
[[.Bold]]Command:[[.Normal]] inject [--pid] [--lhost] <options>
[[.Bold]]About:[[.Normal]] Execute a metasploit payload in a remote process.
[[.Bold]]Options:[[.Normal]]
       --pid | The pid of the process to inject into (required, see 'ps')
   --payload | The MSF payload to use (default: meterpreter_reverse_https)
     --lhost | Metasploit listener LHOST (required)
     --lport | Metasploit listener LPORT (default: 4444)
   --encoder | MSF encoder (default: none)
--iterations | Iterations of the encoder (requires --encoder)

`

	psHelp = `
[[.Bold]]Command:[[.Normal]] ps <options>
[[.Bold]]About:[[.Normal]] List processes on remote system.
[[.Bold]]Options:[[.Normal]]
  --pid | Filter results based on pid
  --exe | Filter results based on exe name (prefix)
--owner | Filter results based on owner (prefix)

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
[[.Bold]]Command:[[.Normal]] cd [dir]
[[.Bold]]About:[[.Normal]] Change working directory.

`

	pwdHelp = `
[[.Bold]]Command:[[.Normal]] pwd
[[.Bold]]About:[[.Normal]] Print working directory.

`

	mkdirHelp = `
[[.Bold]]Command:[[.Normal]] mkdir <remote path> 
[[.Bold]]About:[[.Normal]] Create a remote directory.

`

	rmHelp = `
[[.Bold]]Command:[[.Normal]] rm <remote file> 
[[.Bold]]About:[[.Normal]] Delete a remote file or directory.

`

	catHelp = `
[[.Bold]]Command:[[.Normal]] cat <remote file> 
[[.Bold]]About:[[.Normal]] Cat a remote file to stdout.

`

	downloadHelp = `
[[.Bold]]Command:[[.Normal]] download <remote src> <local dst>
[[.Bold]]About:[[.Normal]] Download a file from the remote system.

`

	uploadHelp = `
[[.Bold]]Command:[[.Normal]] upload <local src> <remote dst>
[[.Bold]]About:[[.Normal]] Upload a file to the remote system.

`

	procdumpHelp = `
[[.Bold]]Command:[[.Normal]] procdump <pid>
[[.Bold]]About:[[.Normal]] Dumps the process memory given a process identifier (pid)

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
