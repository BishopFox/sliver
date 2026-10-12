Sliver supports running in daemon mode, which automatically starts the multiplayer listener but does not provide an interactive console. To connect to a daemonized server, use [multiplayer mode](/docs?name=Multi-player+Mode) operator configs.

By default, daemon mode exposes the multiplayer listener directly over TCP gRPC/mTLS on port `31337`. The multiplayer WireGuard wrapper is opt-in with `--enable-wg` or `daemon.enable_wg: true`.

The daemon also serves a minimal health endpoint for liveness probes (systemd, Docker, Kubernetes) at `GET /healthz` on `127.0.0.1:31339` by default. It responds with `{"status":"ok"}` and discloses no version or build information. The endpoint is bound to loopback by default; a non-loopback address is allowed but logs a warning. Set `--health-addr off` or `daemon.health_addr: off` to disable it.

There are two ways to run in daemon mode:

1. Start daemon directly from CLI: `sliver-server daemon`
2. Set `daemon_mode: true` in `~/.sliver/configs/server.yaml`, then start normally with `sliver-server`

If `server.yaml` does not exist, Sliver generates it. If a legacy `server.json` exists, Sliver migrates it to YAML.

### CLI Behavior

- `sliver-server daemon` supports:
  - `-l, --lhost` multiplayer listener host
  - `-p, --lport` multiplayer listener port
  - `-t, --tailscale` enable Tailscale listener
  - `--enable-wg` wrap multiplayer in WireGuard instead of exposing it directly over mTLS
  - `--health-addr` health endpoint listen address (`host:port`), or `off` to disable
  - `-f, --force` force unpack static assets
- For `sliver-server daemon`, `--lhost` and `--lport` override config values. If omitted, Sliver uses `daemon.host` and `daemon.port` from `server.yaml`.
- For `sliver-server daemon`, `--health-addr` overrides `daemon.health_addr`. If omitted, Sliver uses the config value, which defaults to `127.0.0.1:31339`.
- For normal startup (`sliver-server`) with `daemon_mode: true`, Sliver uses `daemon.tailscale` and `daemon.enable_wg` from `server.yaml`.
- With `--enable-wg` or `daemon.enable_wg: true`, multiplayer is wrapped in WireGuard. Otherwise it stays on direct TCP mTLS.

### Example Config

```yaml
daemon_mode: true
daemon:
    host: ""
    port: 31337
    tailscale: false
    enable_wg: false
    health_addr: 127.0.0.1:31339
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

The `operator` CLI generates direct multiplayer profiles by default. When the daemon is running with the WireGuard wrapper, add `--enable-wg` while generating the profile. That profile contains `wg.enabled: true`, so `sliver-client` selects WireGuard automatically. Use the client's `--enable-wg` or `--disable-wg` only when you need to override the profile.

### Shutdown Behavior

On Unix-like systems, daemon mode handles `SIGTERM` and exits cleanly. The health endpoint drains in-flight requests (up to five seconds) before the listener closes.

### systemd

With daemon mode enabled in `server.yaml`, you can run Sliver under [systemd](https://www.linode.com/docs/quick-answers/linux/start-service-at-boot/) or another init system. See the [Linux install script](/docs?name=Linux+Install+Script) for an example.
