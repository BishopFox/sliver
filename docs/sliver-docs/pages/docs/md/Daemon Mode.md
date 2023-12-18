Starting in v1.0.0 Sliver supports running in "daemon mode," which automatically starts a client listener (but not an interactive console). In order to connect to a server running in daemon mode you'll need to use [multiplayer mode](/docs?name=Multi-player+Mode).

There are two ways to start the server in daemon mode:

1. Using the command line interface: `sliver-server daemon`
2. Set `daemon_mode` to `true` in the `configs/server.json` located in `SLIVER_ROOT_DIR` (by default: `~/.sliver/configs/server.json`). The listener can be configured via the `daemon` object. The process will respond gracefully to `SIGTERM` on Unix-like operating systems.

#### Example Config

```
$ cat ~/.sliver/configs/server.json
{
    "daemon_mode": true,
    "daemon": {
        "host": "",
        "port": 31337
    },
    "logs": {
        "level": 5,
        "grpc_unary_payloads": false,
        "grpc_stream_payloads": false
    }
}
```

#### systemd

With this config you can easily setup a [systemd service](https://www.linode.com/docs/quick-answers/linux/start-service-at-boot/) or init script. See the [Linux install script](/docs?name=Linux+Install+Script) for an example.
