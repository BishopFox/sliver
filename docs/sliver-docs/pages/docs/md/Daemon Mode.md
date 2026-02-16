Sliver supports running in daemon mode, which automatically starts the multiplayer client listener (but not an interactive console). To connect to a daemonized server, use [multiplayer mode](/docs?name=Multi-player+Mode) operator configs.

There are two ways to run in daemon mode:

1. Start daemon directly from CLI: `sliver-server daemon`
2. Set `daemon_mode: true` in `~/.sliver/configs/server.yaml`, then start normally with `sliver-server`

If `server.yaml` does not exist, Sliver generates it. If a legacy `server.json` exists, Sliver migrates it to YAML.

### CLI Behavior

- `sliver-server daemon` supports:
  - `-l, --lhost` multiplayer listener host
  - `-p, --lport` multiplayer listener port
  - `-t, --tailscale` enable Tailscale listener
  - `-f, --force` force unpack static assets
- For `sliver-server daemon`, `--lhost` and `--lport` override config values. If omitted, Sliver uses `daemon.host` and `daemon.port` from `server.yaml`.
- For normal startup (`sliver-server`) with `daemon_mode: true`, Sliver uses `daemon.tailscale` from `server.yaml`.

### Example Config

```yaml
daemon_mode: true
daemon:
    host: ""
    port: 31337
    tailscale: false
logs:
    level: 4
    grpc_unary_payloads: false
    grpc_stream_payloads: false
    tls_key_logger: false
grpc:
    keepalive:
        min_time_seconds: 30
        permit_without_stream: true
```

### Generating Operator Configs Without Console

Since daemon mode does not provide an interactive server console, generate operator configs using the server CLI:

```bash
./sliver-server operator --name zer0cool --lhost 1.2.3.4 --permissions all --save zer0cool.cfg
```

### Shutdown Behavior

On Unix-like systems, daemon mode handles `SIGTERM` and exits cleanly.

### systemd

With daemon mode enabled in `server.yaml`, you can run Sliver under [systemd](https://www.linode.com/docs/quick-answers/linux/start-service-at-boot/) or another init system. See the [Linux install script](/docs?name=Linux+Install+Script) for an example.
