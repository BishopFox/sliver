This script installs the latest version of Sliver, installs Windows cross-compiler dependencies (mingw), and sets up multiplayer for all local users. When systemd or OpenRC is running, it configures Sliver as a managed service. In a container without a running init system it installs the binaries and configuration without attempting to create or start a service.

https://sliver.sh/install

This script should work on Kali, Ubuntu, RHEL (CentOS, etc), and Alpine distributions of Linux. Alpine dependencies are installed with `apk`. The installer uses the init system that is actually running, so merely having another package manager on `PATH` does not change service selection.

**⚠️ OPSEC:** By default the Linux install script will bind the multiplayer listener to `:31337` on all interfaces. In current releases that is the WireGuard-protected multiplayer listener, so the outer service is UDP/31337 and the authenticated gRPC/mTLS server only exists inside the tunnel. Ensure your firewalls are properly configured if this is a concern, or reconfigure the server to bind to localhost if you only wish to allow local users. Publicly exposing the multiplayer listener still makes the server easier to discover and fingerprint.

### One Liner

On distributions with `curl` and `sudo`:

```
curl -fsSL https://sliver.sh/install | sudo sh
```

A stock Alpine installation includes BusyBox `wget`. Run the installer from a root shell:

```
wget -qO- https://sliver.sh/install | sh
```

- Installs server binary to `/root/sliver-server`
- Installs mingw
- Runs the server in daemon mode when an active systemd or OpenRC instance is detected
- Installs client to `/usr/local/bin/sliver`
- Generates multiplayer configurations for all users with a `/home` directory

Containers typically do not run an init system. In that case the installer completes without registering a service and prints the foreground command to use as the container entrypoint:

```
exec /root/sliver-server daemon
```

### Systemd Service

The following systemd configuration is used:

```ini
[Unit]
Description=Sliver
After=network.target
StartLimitIntervalSec=0

[Service]
Type=simple
Restart=on-failure
RestartSec=3
User=root
ExecStart=/root/sliver-server daemon

[Install]
WantedBy=multi-user.target
```

### OpenRC Service

When OpenRC is running, the installer creates a supervised service that restarts Sliver after an unexpected exit:

```sh
#!/sbin/openrc-run
name="sliver"
description="Sliver server"
command="/root/sliver-server"
command_args="daemon"
supervisor=supervise-daemon
pidfile="/run/sliver.pid"
respawn_delay=3
respawn_max=0

depend() {
    after net
}
```
