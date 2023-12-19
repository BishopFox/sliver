Sliver supports two types of SOCKS5 proxies, an "in-band" proxy that tunnels though any C2 protocol, and WireGuard SOCKS proxy (only available when using WireGuard C2).

## In-band SOCKS5

An in-band reverse SOCKS5 proxy is supported in Sliver versions 1.5 and later. Note that the SOCKS proxy feature can only be used on sessions (i.e., interactive sessions) and not beacons.

```
sliver (UGLY_SCARIFICATION) > socks5 add

[*] Started SOCKS5 127.0.0.1 1081
⚠️  In-band SOCKS proxies can be a little unstable depending on protocol
```

Simply upstream to `127.0.0.1:1081` from here, see `socks5 add --help` for more options.

## WireGuard SOCKS5

In order to use `wg-socks` you'll need a WireGuard client, any client should work. However, we recommend using `wg-quick`, which is included in the `wireguard-tools` package available on most platforms (see [WireGuard](https://www.wireguard.com/install/) for more platforms):

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

Interact with the session, and use `wg-socks` to create a SOCKS proxy!
