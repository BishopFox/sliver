Sliver
======

![Sliver](/sliver/sliver.jpeg)

Sliver is a remote shellcode loading and injection service that uses end-to-end encryption (mTLS) for all network traffic. Implants are dynamically compiled with unique X.509 certificates signed by a per-instance certificate authority generated when you first run the binary. Sliver can load arbitrary shellcode but also integrates with MSFVenom to automatically generate, execute, and manage Meterpreter payloads. Sliver binaries have very low anti-virus detection as they do not contain any malicous code themselves and instead dynamically load it over the network.

Sliver can inject payloads into it's own process or optionally use remote thread injection to execute payloads in a remote processes to which your execution context has access. 

### Feature Progess

#### C2
- [x] Mutual TLS
- [x] DNS
- [x] HTTP(S)
- [ ] ICMP

#### Chain Loader
- [x] Raw Shellcode
- [x] .NET Assemblies
- [x] Metasploit/Meterpreter (v5 or later)
- [ ] Empire
- [ ] Cobalt Strike

#### Post Exploitation
- [x] Windows Token Manipulation
- [x] Procdump
- [ ] TCP tunnels
- [ ] Reverse SOCKS proxy  

### Setup

1. Download the latest [Sliver](https://github.com/BishopFox/sliver/releases) binary
2. ???
3. Shellz

## Compile From Source

See the [wiki](https://github.com/BishopFox/sliver/wiki/Compile-From-Source)