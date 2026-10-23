# Sliver

Sliver is an open-source, cross-platform adversary emulation and red team framework designed for organizations of all sizes to perform authorized security testing.

Sliver implants support command-and-control (C2) over Mutual TLS (mTLS), WireGuard, HTTP(S), and DNS. Implants are dynamically compiled with unique per-binary asymmetric encryption keys and support a wide range of post-exploitation capabilities across macOS, Windows, and Linux.

Sliver includes support for native and shellcode payloads, userspace reflective loading, built-in shellcode encoding, and BOF/COFF execution on both amd64 and arm64 architectures for macOS, Windows, and Linux.

The Sliver server and client run on macOS, Windows, and Linux. Implants officially support those same platforms and may also run on additional targets supported by the Go compiler, although those configurations are not regularly tested.

[![Release](https://github.com/BishopFox/sliver/actions/workflows/autorelease.yml/badge.svg)](https://github.com/BishopFox/sliver/actions/workflows/autorelease.yml) [![golangci-lint](https://github.com/BishopFox/sliver/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/BishopFox/sliver/actions/workflows/golangci-lint.yml) [![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

Visit [https://sliver.sh/](https://sliver.sh/) for tutorials and documentation.

### Getting Started

Download the latest [release](https://github.com/BishopFox/sliver/releases) and see the Sliver [wiki](https://sliver.sh/docs?name=Getting+Started) for a quick tutorial on basic setup and usage. To get the very latest and greatest compile from source.

#### Linux One Liner

`curl https://sliver.sh/install|sudo bash` and then run `sliver`

### Help!

Please checkout the [wiki](https://sliver.sh/), or start a [GitHub discussion](https://github.com/BishopFox/sliver/discussions).

### Compile From Source

See the [wiki](https://sliver.sh/docs?name=Compile+from+Source).

### License - GPLv3

Sliver is licensed under [GPLv3](https://www.gnu.org/licenses/gpl-3.0.en.html), some sub-components may have separate licenses. See their respective subdirectories in this project for details.
