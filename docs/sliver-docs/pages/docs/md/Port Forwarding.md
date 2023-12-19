Sliver provides two mechanisms for port forwarding to tunnel additional connections / tools into the target environment via an implant:

- `portfwd` - This command (available on all C2s) uses Sliver's in-band tunnels to transfer data between your local machine and the implant network (i.e., if you're using HTTP C2 all port forwarded traffic is tunneled over HTTP, same for mTLS/DNS/etc.)
- `wg-portfwd` - This command uses WireGuard port forwarding, and is only available when using WireGuard C2.

**NOTE:** Generally speaking `wg-portfwd` is faster and more reliable, we recommend using it whenever possible. Some protocols may be unstable, or may not work when tunneled via `portfwd`. However, `wg-portfwd` does requires a little extra setup (see below).

## In-Band Tunneled Port Forwarding

Tunneled port forwarding can be done over any C2 transport, and should work out of the box. Interact with the session you'd like to port forward through and use the `portfwd add` command:

```
sliver (STUCK_ARTICLE) > portfwd add --remote 10.10.10.10:22

[*] Port forwarding 127.0.0.1:8080 -> 10.10.10.10:22
```

By default all port forwards will be bound to the `127.0.0.1` interface, but you can override this using the `--bind` flag. Port forwarding also works in `multiplayer` mode and will forward ports to your local system.

#### Reverse Port Forwarding

As of v1.5.27 Sliver also supports reverse port forwarding via the `rportfwd` command.

## WireGuard Port Forwarding

In order to use `wg-portfwd` you'll need a WireGuard client, any client should work. However, we recommend using `wg-quick`, which is included in the `wireguard-tools` package available on most platforms (see [WireGuard](https://www.wireguard.com/install/) for more platforms):

- MacOS `brew install wireguard-tools`
- Ubuntu/Kali `sudo apt install wireguard-tools`

First generate a WireGuard C2 implant (using `generate --wg`), and then start a WireGuard listener:

```
sliver > wg

[*] Starting Wireguard listener ...
[*] Successfully started job #1

sliver > jobs

ID  Name  Protocol  Port
==  ====  ========  ====
1   wg    udp       53
```

Next, using Sliver you can create WireGuard client configuration using the `wg-config` command (you can use `--save` to write the configuration directly to a file):

```
sliver > wg-config

[*] New client config:
[Interface]
Address = 100.64.0.16/16
ListenPort = 51902
PrivateKey = eMdqQ5zEF9Oflj+7wfyFQZjES02rfSBfZEN701FzmmQ=
MTU = 1420

[Peer]
PublicKey = HNFS0FydHkuCtEFPPFb3b2IW7iSmFajRJ2qSjifidiM=
AllowedIPs = 100.64.0.0/16
Endpoint = <configure yourself>
```

The only thing in the configuration file you'll need to change is the `Endpoint` setting, configure this to point to the Sliver server's WireGuard listener, and ensure to include the port number (by default UDP 53). Generally this will be the same value you specified as `--lhost` when generating the binary.

Make sure your WireGuard listener is running and connect using the client configuration:

```
$ wg-quick up wireguard.conf
```

Now that your machine is connected to the Sliver WireGuard listener, just wait for an implant to connect:

```
sliver > sessions

ID  Name           Transport  Remote Address     Hostname     Username  Operating System  Last Check-in                  Health
==  ====           =========  ==============     ========     ========  ================  =============                  ======
1   STUCK_ARTICLE  wg         100.64.0.17:53565  MacBook-Pro  jdoe      darwin/amd64      Wed, 12 Apr 2021 19:21:00 CDT  [ALIVE]
```

Interact with the session, and use `wg-portfwd add` to create port forwards:

```
sliver (STUCK_ARTICLE) > wg-portfwd add --remote 10.10.10.10:3389

[*] Port forwarding 100.64.0.17:1080 -> 10.10.10.10:3389
```

You can now connect to `100.64.0.17:1080` via your WireGuard interface and the connection will come out at `10.10.10.10:3389`!
