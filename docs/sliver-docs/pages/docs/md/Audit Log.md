Sliver keeps an audit log of every command and its arguments executed by the server (including commands executed by operators in multiplayer mode), as well as most events (such as a new session or beacon connecting to the server). The audit log's intended use is for after-action analysis; providing a detailed history of the entire engagement, including which commands were executed on which hosts when. This will include any commands executed by any operator on any session. Note some console commands only perform actions on the "client-side" and may not appear in the audit log, but will still appear in the client's command history. Additionally, interactive commands (e.g., `shell`) may not appear in the logs aside from the initial usage of the `shell` command.

By default the audit log is located on the server at: `~/.sliver/logs/audit.json`. However, this can be changed by modifying the [`SLIVER_ROOT_DIR`](/docs?name=Environment+Variables) environment variable.

#### Parsing Audit Logs

The audit log is stored in a newline delimited (one object per line) nested-JSON format designed to be primarily machine readable, an example entry is shown below:

```
{"level":"info","msg":"{\"request\":\"{\\\"Port\\\":8888}\",\"method\":\"/rpcpb.SliverRPC/StartMTLSListener\"}","time":"2021-06-16T10:22:54-05:00"}
```

**NOTE:** Due to limitations in the logging APIs the audit log contains nested JSON objects that may require additional parsing.

The top level JSON should always contain:

- `level` - The level indicates the type of action performed. Currently, `info` indicates commands and `warn` indicates events.
- `msg` - A JSON object encoded as a string. This object should always contain a `request` and a `method`. The contents of `method` indicate which command was executed, `request` will contain parameters to that command. The contents of `request` will vary depending on the command, but it will be based on the corresponding gRPC/Protobuf message.
- `time` - The server's timestamp of the log entry
