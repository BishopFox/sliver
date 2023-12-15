This script will install the latest version of Sliver as a systemd service, installs Windows cross-compiler dependencies (mingw), and setup multiplayer for the all local users. After running the script connect locally by running `sliver`.

https://sliver.sh/install

This script should work on Kali, Ubuntu, and RHEL (CentOS, etc) distributions of Linux.

**⚠️ OPSEC:** By default the Linux install script will bind the multiplayer listener to `:31337` i.e. all interfaces. Ensure your firewalls are properly configured if this is a concern, or reconfigure the server to bind to localhost if you only wish to allow local users. Multiplayer Mode is authenticated using mTLS/gRPC and per-operator bearer tokens, so this doesn't present a security problem (notwithstanding bugs in our implementation), but publicly exposing your multiplayer listener will make it trivial to identify what the server is running.

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
