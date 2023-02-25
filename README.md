Sliver
======

Sliver is an open source cross-platform adversary emulation/red team framework, it can be used by organizations of all sizes to perform security testing. Sliver's implants support C2 over Mutual TLS (mTLS), WireGuard, HTTP(S), and DNS and are dynamically compiled with per-binary asymmetric encryption keys.

The server and client support MacOS, Windows, and Linux. Implants are supported on MacOS, Windows, and Linux (and possibly every Golang compiler target but we've not tested them all).

[![Release](https://github.com/BishopFox/sliver/actions/workflows/autorelease.yml/badge.svg)](https://github.com/BishopFox/sliver/actions/workflows/autorelease.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/BishopFox/sliver)](https://goreportcard.com/report/github.com/BishopFox/sliver) [![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

### Features

* Dynamic code generation
* Compile-time obfuscation
* Multiplayer-mode
* Staged and Stageless payloads
* [Procedurally generated C2](https://github.com/BishopFox/sliver/wiki/HTTP(S)-C2#under-the-hood) over HTTP(S)
* [DNS canary](https://github.com/BishopFox/sliver/wiki/DNS-C2#dns-canaries) blue team detection
* [Secure C2](https://github.com/BishopFox/sliver/wiki/Transport-Encryption) over mTLS, WireGuard, HTTP(S), and DNS
* Fully scriptable using [JavaScript/TypeScript](https://github.com/moloch--/sliver-script) or [Python](https://github.com/moloch--/sliver-py)
* Windows process migration, process injection, user token manipulation, etc.
* Let's Encrypt integration
* In-memory .NET assembly execution
* COFF/BOF in-memory loader
* TCP and named pipe pivots
* Much more!

### Getting Started

Download the latest [release](https://github.com/BishopFox/sliver/releases) and see the Sliver [wiki](https://github.com/BishopFox/sliver/wiki/Getting-Started) for a quick tutorial on basic setup and usage. To get the very latest and greatest compile from source.

#### Linux One Liner

`curl https://sliver.sh/install|sudo bash` and then run `sliver`

### Help!

Please checkout the [wiki](https://github.com/BishopFox/sliver/wiki), or start a [GitHub discussion](https://github.com/BishopFox/sliver/discussions). We also tend to hang out in the #golang Slack channel on the [Bloodhound Gang](https://bloodhoundgang.herokuapp.com/) server.

### Compile From Source

See the [wiki](https://github.com/BishopFox/sliver/wiki/Compile-From-Source).

### Feedback

Please take a moment and fill out [our survey](https://forms.gle/SwVsHFNh24ChG58C6)

### License - GPLv3

Sliver is licensed under [GPLv3](https://www.gnu.org/licenses/gpl-3.0.en.html), some sub-components may have separate licenses. See their respective subdirectories in this project for details.

