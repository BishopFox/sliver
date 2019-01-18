Sliver
======

Sliver is a remote shellcode loading and injection service that uses end-to-end encryption (mTLS) for all network traffic. Implants are dynamically compiled with unique X.509 certificates signed by a per-instance certificate authority generated when you first run the binary. Sliver can load arbitrary shellcode but also integrates with MSFVenom to automatically generate, execute, and manage Meterpreter payloads. Sliver binaries have very low anti-virus detection as they do not contain any malicous code themselves and instead dynamically load it over the network.

Sliver can inject payloads into it's own process or optionally use remote thread injection to execute payloads in a remote processes to which your execution context has access to.

```
[Attacker] <-(mTLS)-> [sliver] -(code injection)-> [remote process]
```

## Building From Scratch ##

You'll want to compile from a MacOS or Linux machine, compiling from Windows should work but none of the scripts are designed to run on Windows (you can compile the Windows binaries from MacOS or Linux).

Requirements:

* Go v1.11 or later

* Make, sed, tar, wget, zip

Build thin server (for developement)

```
$ ./deps.sh
$ ./go-assets.sh
$ make
```

Statically compile and bundle server with all dependencies and assets:

```
$ make static-macos
$ make static-linux
$ make static-windows
```
