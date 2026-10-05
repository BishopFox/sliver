Sliver's audit log records server-side unary gRPC requests, including multiplayer operator requests, plus session and beacon registrations. It supports after-action analysis, but request entries are written before execution and do not confirm success. Client-only actions and interactive command input (e.g., within `shell`) are not recorded in the audit log.

By default the audit log is located on the server at: `~/.sliver/logs/audit.json`. However, this can be changed by modifying the [`SLIVER_ROOT_DIR`](/docs?name=Environment+Variables) environment variable.

By default, logs rotate at 50 MB and retain up to 20 compressed backups for at most 90 days. These limits are configurable under `logs` in `server.yaml`.

#### Parsing Audit Logs

The audit log uses a newline delimited (one object per line), nested-JSON format. A simplified RPC entry is shown below:

```
{"level":"info","msg":"{\"request\":\"{\\\"Port\\\":8888}\",\"method\":\"/rpcpb.SliverRPC/StartMTLSListener\"}","time":"2021-06-16T10:22:54-05:00"}
```

**NOTE:** Due to limitations in the logging APIs the audit log contains nested JSON objects that may require additional parsing.

The top level JSON should always contain:

- `level` - Currently, `info` indicates RPC requests and `warn` indicates registration events.
- `msg` - A JSON object encoded as a string. RPC entries contain `request` (another JSON-encoded string), `method`, `user`, `remote_ip`, and optional `session` or `beacon` details. Registration events instead contain `Session` or `Beacon` and `Register`, without `request` or `method`.
- `time` - The server's timestamp of the log entry
