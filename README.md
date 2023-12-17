# Sliver

Sliver is an open source cross-platform adversary emulation/red team framework, it can be used by organizations of all sizes to perform security testing. Sliver's implants support C2 over Mutual TLS (mTLS), WireGuard, HTTP(S), and DNS and are dynamically compiled with per-binary asymmetric encryption keys.

The server and client support MacOS, Windows, and Linux. Implants are supported on MacOS, Windows, and Linux (and possibly every Golang compiler target but we've not tested them all).

[![Release](https://github.com/BishopFox/sliver/actions/workflows/autorelease.yml/badge.svg)](https://github.com/BishopFox/sliver/actions/workflows/autorelease.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/BishopFox/sliver)](https://goreportcard.com/report/github.com/BishopFox/sliver) [![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

# v1.6.0 / `master`

**NOTE:** You are looking the latest master branch of Sliver v1.6.0; new PRs should target this branch. However, this branch is NOT RECOMMENDED for production use yet. Please use release tagged versions for the best experience.

For PRs containing bug fixes specific to Sliver v1.5, please target the [`v1.5.x/master` branch](https://github.com/BishopFox/sliver/tree/v1.5.x/master).

### Features

- Dynamic code generation
- Compile-time obfuscation
- Multiplayer-mode
- Staged and Stageless payloads
- [Procedurally generated C2](https://sliver.sh/docs?name=HTTPS+C2) over HTTP(S)
- [DNS canary](https://sliver.sh/docs?name=DNS+C2) blue team detection
- [Secure C2](https://sliver.sh/docs?name=Transport+Encryption) over mTLS, WireGuard, HTTP(S), and DNS
- Fully scriptable using [JavaScript/TypeScript](https://github.com/moloch--/sliver-script) or [Python](https://github.com/moloch--/sliver-py)
- Windows process migration, process injection, user token manipulation, etc.
- Let's Encrypt integration
- In-memory .NET assembly execution
- COFF/BOF in-memory loader
- TCP and named pipe pivots
- Much more!

### Getting Started

Download the latest [release](https://github.com/BishopFox/sliver/releases) and see the Sliver [wiki](https://sliver.sh/docs?name=Getting+Started) for a quick tutorial on basic setup and usage. To get the very latest and greatest compile from source.

#### Linux One Liner

`curl https://sliver.sh/install|sudo bash` and then run `sliver`

### Help!

Please checkout the [wiki](https://sliver.sh/), or start a [GitHub discussion](https://github.com/BishopFox/sliver/discussions). We also tend to hang out in the #golang Slack channel on the [Bloodhound Gang](https://bloodhoundgang.herokuapp.com/) server.

### Compile From Source

See the [wiki](https://sliver.sh/docs?name=Compile+from+Source).

### Feedback

Please take a moment and fill out [our survey](https://forms.gle/SwVsHFNh24ChG58C6)

### License - GPLv3

Sliver is licensed under [GPLv3](https://www.gnu.org/licenses/gpl-3.0.en.html), some sub-components may have separate licenses. See their respective subdirectories in this project for details.
