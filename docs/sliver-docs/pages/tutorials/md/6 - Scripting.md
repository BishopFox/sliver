# This course is intended for the 1.6 version of Sliver, which is not yet published

Reactions are a basic way to automate tasks in the sliver console, they allow you to specify sliver commands to run on a list of events.

```bash
Reactable Events:
   session-connected  Triggered when a new session is opened to a target
     session-updated  Triggered on changes to session metadata
session-disconnected  Triggered when a session is closed (for any reason)
              canary  Triggered when a canary is burned or created
          watchtower  Triggered when implants are discovered on threat intel platforms
          loot-added  Triggered when a new piece of loot is added to the server
        loot-removed  Triggered when a piece of loot is removed from the server
```

Let’s go ahead and create a reaction to list the current directory and environment variables when a new session checks in.

```bash
reaction set -e "session-connected"

[*] Setting reaction to: Session Opened

? Enter commands:  [Enter 2 empty lines to finish]pwd
env
? Enter commands:
pwd
env

[*] Set reaction to session-connected (id: 1)
```

The reaction is now set, if you spin up a new session these commands will be automatically run on that session’s initial connection.

```bash
[*] Session 99c7a639 UNEXPECTED_PORTER - 127.0.0.1:59966 (test.local) - darwin/amd64 - Thu, 04 May 2023 09:04:58 CEST

[*] Execute reaction: 'pwd'

[*] /Users/tester

[*] Execute reaction: 'env'

PWD=/Users/tester
COLORTERM=truecolor
...
```

You can remove reactions using `reaction unset`.

However, there are a couple of limitations to keep in mind when using reactions, first off, these are run in the console you are currently using, which is not necessarily the server console. So if you are connected to a sliver server using the sliver client, if you disconnect the client the reactions are no longer running. 

Secondly reactions are a relatively basic mechanism, you can’t use any conditional statements or more complex background tasks with them. For more complex use-cases you can instead write your own client in Python or Typescript for example to connect to the server over gRPC, which we’ll cover next.

## Sliver-py

For the purposes of this tutorial we’ll write our extensions using Python3, however the same result is achievable using Typescript, Golang or any other language that can handle gRPC.

First, install the sliver-py extension using pip.

```bash
pip3 install sliver-py
```

Since our extension is essentially going to be another client connection to the sliver server, you’ll also need to enable multiplayer mode and generate a new profile

```bash
[server] sliver > multiplayer

[*] Multiplayer mode enabled!

[server] sliver > new-operator -n tester -l 127.0.0.1

[*] Generating new client certificate, please wait ...
[*] Saved new client config to: /Users/tester/tools/tester_127.0.0.1.cfg
```

We now have everything we need to start writing our scripts, let’s run our first example interactively in a Python shell. 
We first need to import a few dependencies, `SliverClientConfig`, which is used to parse the client config we’ve just created, and `SliverClient`, which will handle the connection to the backend server.

```bash
Python 3.9.16 (main, Dec  7 2022, 10:06:04)
Type 'copyright', 'credits' or 'license' for more information
IPython 8.0.1 -- An enhanced Interactive Python. Type '?' for help.

In [1]: from sliver import SliverClientConfig, SliverClient

In [2]: DEFAULT_CONFIG = "/Users/tester/tools/tester_127.0.0.1.cfg"

In [3]: config = SliverClientConfig.parse_config_file(DEFAULT_CONFIG)

In [4]: client = SliverClient(config)

In [5]: await client.connect()
Out[5]:
Major: 1
Minor: 5
Patch: 37
Commit: "0a43dc688ffb31a0a38511c47e8547a44a6918d4"
CompiledAt: 1681408237
OS: "darwin"
Arch: "arm64"
```

From this point on we can use the client object to interact with the server, let’s start with listing any sessions or beacons that might be currently connected.

```bash
In [6]: beacons = await client.beacons()

In [7]: sessions = await client.sessions()

In [8]: beacons
Out[8]: []

In [9]: sessions
Out[9]:
[ID: "f80ec897-0870-4f03-a1b1-364e5a0d243c"
 Name: "UNEXPECTED_PORTER"
 Hostname: "test.local"
 UUID: "c6de1a44-016a-5fbe-b76a-da56af41316d"
 Username: "tester"
 UID: "501"
 GID: "20"
 OS: "darwin"
 Arch: "amd64"
 Transport: "http(s)"
 RemoteAddress: "127.0.0.1:60218"
 PID: 74773
 Filename: "/Users/tester/tools/UNEXPECTED_PORTER"
 LastCheckin: 1683185925
 ActiveC2: "http://127.0.0.1"
 ReconnectInterval: 60000000000
 PeerID: 4416183373589698218
 FirstContact: 1683185429]
```

To run commands on this session you’ll need to create an InteractiveSession object.

```bash
In [10]: interact = await client.interact_session("f80ec897-0870-4f03-a1b1-364e5a0d243c")

In [11]: await interact.pwd()
Out[11]: Path: "/Users/tester"
```

Now that we’ve got the basics of connecting to sliver and running commands down let’s write a more useful script that will display the hosts file when a new session checks in. Our goal is to first identify the Operating System, and then based on that retrieve and display the contents of the hosts file if it exists. Because this script will wait and react to events emitted by the Sliver server, we’re going to use `asyncio` to write our client.

```bash
#!/usr/bin/env python3

import os
import asyncio
from sliver import SliverClientConfig, SliverClient
import gzip

DEFAULT_CONFIG = "/Users/tester/tools/neo_127.0.0.1.cfg"

async def main():
    ''' Client connect example '''
    config = SliverClientConfig.parse_config_file(DEFAULT_CONFIG)
    client = SliverClient(config)
    await client.connect()

    async for event in client.on('session-connected'):
        print('Session %s just connected !' % event.Session.ID)

if __name__ == '__main__':
    loop = asyncio.get_event_loop()
    loop.run_until_complete(main())
```

As shown above we can access the session object through `event.Session`. Let’s go ahead and add a few conditions based on the operating system.

```bash
if event.Session.OS == "darwin":
            print('Session is running on macOS')

elif event.Session.OS == "Linux":
            print('Session is running on Linux')
elif event.Session.OS == "Windows"
            print('Session is running on Windows')
else:
            print('Session is running on %s', event.Session.OS)
```

Let’s set up an InteractiveSession object like previously.

```bash
interact = await client.interact_session(event.Session.ID)
```

We’re going to start with writing the code for Linux and macOS, since in their case the file is located in the same place. First we check if the file exists, then we download and decompress it to display its contents using gzip.

```bash
file_listing = await interact.ls("/etc/hosts")
if file_listing.Exists:
	gzipFile = await interact.download("/etc/hosts")
  contents = gzip.decompress(gzipFile.Data)
  print('%r' % contents)
```

The code for Windows is relatively similar the only major difference being the file location.

```bash
file_listing = await interact.ls("C:/Windows/System32/drivers/etc/hosts")
if file_listing.Exists:
	gzipFile = await interact.download("C:/Windows/System32/drivers/etc/hosts")
  contents = gzip.decompress(gzipFile.Data)
  print('%r' % contents)
```

If we run our script and spin up a few sessions we should start to see hosts files being retrieved.

```bash
python3.11 autocat.py
Automatically interacting with session 16338c85-b670-44ab-ac83-2df885654b07
b"# Copyright (c) 1993-2009 Microsoft Corp.\r\n#\r\n# This is a sample HOSTS file used by Microsoft TCP/IP for Windows.\r\n#\r\n# ...

Automatically interacting with session 93fcbab2-f00d-44a4-944a-e1ea8ec324e2
b'##\n# Host Database\n#\n# localhost is used to configure the loopback interface\n# when the system is booting.  Do not change this entry.\n##\n127.0.0.1...
```
