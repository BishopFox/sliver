This script installs the latest version of Sliver as a systemd service, installs Windows cross-compiler dependencies (mingw), and sets up multiplayer for all local users. After running the script, connect locally by running `sliver`.

https://sliver.sh/install

This script should work on Kali, Ubuntu, and RHEL (CentOS, etc) distributions of Linux.

**⚠️ OPSEC:** By default the Linux install script binds the direct multiplayer gRPC/mTLS listener to TCP `:31337` on all interfaces. WireGuard is not enabled by the installer. Ensure your firewalls are properly configured, or reconfigure the server to bind to localhost if you only wish to allow local users. Publicly exposing the multiplayer listener makes the server easier to discover and fingerprint.

### One Liner

```
curl https://sliver.sh/install|sudo bash
```

- Installs server binary to `/root/sliver-server`
- Installs mingw
- Runs the server in daemon mode using systemd
- Installs client to `/usr/local/bin/sliver`
- Generates multiplayer configurations for all users with a `/home` directory

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
