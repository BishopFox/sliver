Sliver
======

Sliver is a remote shellcode loading and injection service that uses end-to-end encryption (mTLS) for all network traffic. Implants are dynamically compiled with unique X.509 certificates signed by a per-instance certificate authority generated when you first run the binary. Sliver can load arbitrary shellcode but also integrates with MSFVenom to automatically generate, executate, and manage Meterpreter payloads.


## Building From Scratch

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
