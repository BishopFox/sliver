__IMPORTANT:__ MCP (Model Context Protocol) support is _experimental_ and NOT ALL FUNCTIONALITY IS SUPPORTED YET. Pull requests and testing feedback are welcome!

There are two main ways to connect an AI model to sliver using MCP: locally over stdio, or over a network using HTTP/SSE transports. The `sliver-client` supports both stdio and http/sse, but the `sliver-server` binary only supports HTTP/SSE. We recommend using the `sliver-client` and [Multiplayer Mode](/docs?name=Multi-player+Mode) to connect to a remote server with the stdio transport.


## MCP Over STDIO

Example configurations for various AIs, you'll need to configure [Multiplayer Mode](/docs?name=Multi-player+Mode) first:

### OpenAI Codex

```
[mcp_servers.sliver]
args = ["mcp", "--config", "/path/to/multiplayer.cfg"]
command = "/path/to/sliver-client"
```

### Anthropic Claude Code

```
claude mcp add sliver -- /path/to/sliver-client mcp --config /path/to/multiplayer.cfg
```

## MCP Over HTTP/SSE

The MCP HTTP/SSE server can be started from either the `sliver-client` or `sliver-server`, however _where you run the command is where the server will start_ i.e., starting the HTTP/SSE MCP using `sliver-client` will turn that `sliver-client` binary into the MCP server (connected to whatever Sliver server you're currently connected to). Running the command from the `sliver-server` console will start a HTTP/SSE MCP server locally bound on the server.

```
.------..------..------..------..------..------.
|S.--. ||L.--. ||I.--. ||V.--. ||E.--. ||R.--. |
| :/\: || :/\: || (\/) || :(): || (\/) || :(): |
| :\/: || (__) || :\/: || ()() || :\/: || ()() |
| '--'S|| '--'L|| '--'I|| '--'V|| '--'E|| '--'R|
`------'`------'`------'`------'`------'`------'

All hackers gain skulk
[*] Server v1.6.4 - 4d0d46afd6e1cbcaafde92687144cb28694a68ce
[*] Welcome to the sliver shell, please type 'help' for options
sliver > mcp

Status: stopped
Transport: sse
Listen: 127.0.0.1:8080
Endpoint: http://127.0.0.1:8080/sse

sliver > mcp start --transport http

[*] Starting MCP server (http) on 127.0.0.1:8080
[*] Endpoint: http://127.0.0.1:8080/mcp
```

You can use `mcp stop` to stop the server, see `mcp start --help` for additional options when starting the MCP server. Authentication is not yet supported for MCP over HTTP/SSE.