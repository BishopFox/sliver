# AI Agent Integration Guide

This guide explains how to integrate AI Agents with Sliver C2 using the MCP (Model Context Protocol) Server and CLI structured output features.

## MCP Server

Sliver provides a built-in MCP Server that allows AI Agents to interact with the C2 framework programmatically.

### Starting the MCP Server

```bash
# Start MCP server with HTTP transport
sliver > mcp start --transport http --listen 127.0.0.1:8080

# Start MCP server with SSE transport
sliver > mcp start --transport sse --listen 127.0.0.1:8080

# Start MCP server via stdio (for direct AI Agent integration)
sliver mcp --stdio
```

### Available MCP Tools

The MCP Server exposes the following tools for AI Agents:

#### Session Management
- **list_sessions_and_beacons** - List all active sessions and beacons
  - Returns: Array of sessions and beacons with their metadata

#### File System Operations
- **fs_ls** - List directory contents on remote target
- **fs_cd** - Change directory on remote target
- **fs_cat** - Download and read file contents from remote target
- **fs_pwd** - Get current working directory on remote target
- **fs_rm** - Remove files or directories on remote target
- **fs_mv** - Move or rename files on remote target
- **fs_cp** - Copy files on remote target
- **fs_mkdir** - Create directories on remote target
- **fs_chmod** - Change file permissions on remote target
- **fs_chown** - Change file ownership on remote target

#### Command Execution
- **execute_command** - Execute commands on remote target
  - Parameters: `command`, `args[]`, `env[]`, `background`, `output`
  - Returns: stdout, stderr, exit status

#### File Transfer
- **upload_file** - Upload files to remote target
  - Parameters: `local_path` or `data_base64`, `remote_path`, `overwrite`
  - Returns: bytes written, remote path

#### System Information
- **get_system_info** - Get detailed system information from remote target
  - Returns: hostname, OS, version, arch, username, PID, locale, active C2, proxy URL

#### Network Operations
- **list_pivots** - List all active pivot listeners and connections
  - Returns: Array of pivot listeners with their configuration

### Authentication

The MCP Server supports token-based authentication:

```bash
# Set auth token via environment variable
export SLIVER_MCP_TOKEN=your-secret-token

# Or configure in mcp.yaml
```

### Example: Using with Claude Desktop

Add to your Claude Desktop configuration:

```json
{
  "mcpServers": {
    "sliver": {
      "command": "sliver",
      "args": ["mcp", "--stdio"],
      "env": {
        "SLIVER_MCP_TOKEN": "your-token"
      }
    }
  }
}
```

## CLI Structured Output

Sliver CLI supports structured output formats for automation and AI Agent integration.

### JSON Output

```bash
# List sessions in JSON format
sliver sessions --output-format json

# Example output:
{
  "sessions": [
    {
      "id": "abc123",
      "name": "target-01",
      "remote_address": "192.168.1.100",
      "hostname": "WORKSTATION",
      "username": "admin",
      "os": "Windows 10",
      "arch": "amd64",
      "pid": 1234,
      "last_checkin": 1704123456
    }
  ],
  "sessions_count": 1
}
```

### YAML Output

```bash
# List sessions in YAML format
sliver sessions --output-format yaml

# Example output:
sessions:
  - id: abc123
    name: target-01
    remote_address: 192.168.1.100
    hostname: WORKSTATION
    username: admin
    os: Windows 10
    arch: amd64
    pid: 1234
    last_checkin: 1704123456
sessions_count: 1
```

### Supported Commands

The following commands support `--output-format` parameter:
- `sessions` - List active sessions
- `beacons` - List active beacons
- `jobs` - List running jobs
- `operators` - List operators
- `loot` - List loot items
- `creds` - List credentials
- `hosts` - List known hosts

## AI Agent Best Practices

### 1. Session Management

```python
# Example: List sessions and select one
sessions = mcp_client.call_tool("list_sessions_and_beacons", {})
active_session = sessions["sessions"][0]
session_id = active_session["id"]
```

### 2. Command Execution

```python
# Execute a command and get output
result = mcp_client.call_tool("execute_command", {
    "session_id": session_id,
    "command": "whoami",
    "args": [],
    "output": True
})
print(result["stdout"])
```

### 3. File Operations

```python
# Upload a file
upload_result = mcp_client.call_tool("upload_file", {
    "session_id": session_id,
    "local_path": "/path/to/local/file",
    "remote_path": "/tmp/uploaded_file"
})

# Download a file
download_result = mcp_client.call_tool("fs_cat", {
    "session_id": session_id,
    "path": "/etc/passwd"
})
file_content = base64.b64decode(download_result["data_base64"])
```

### 4. System Reconnaissance

```python
# Get system information
sysinfo = mcp_client.call_tool("get_system_info", {
    "session_id": session_id
})
print(f"OS: {sysinfo['os']}, Arch: {sysinfo['arch']}")
```

### 5. Beacon Operations (Async)

```python
# For beacons, operations are async
result = mcp_client.call_tool("execute_command", {
    "beacon_id": beacon_id,
    "command": "whoami",
    "wait": True,  # Wait for result
    "timeout_seconds": 60
})

# Or check status later
if result["async"]:
    task_id = result["task_id"]
    # Poll for completion or use task ID to retrieve results
```

## Security Considerations

1. **Authentication**: Always use authentication tokens for MCP Server
2. **Network Isolation**: Bind MCP Server to localhost unless remote access is required
3. **Token Rotation**: Regularly rotate MCP authentication tokens
4. **Audit Logging**: Monitor MCP Server logs for unauthorized access attempts
5. **Least Privilege**: Grant AI Agents only the permissions they need

## Troubleshooting

### MCP Server won't start
- Check if port is already in use
- Verify Sliver server is running and accessible
- Check logs with `sliver mcp` to see current status

### Authentication failures
- Verify SLIVER_MCP_TOKEN environment variable is set
- Check mcp.yaml configuration file
- Ensure token matches between client and server

### Command execution timeouts
- Increase `timeout_seconds` parameter for long-running commands
- Use `background: true` for commands that don't need immediate output
- Check network connectivity to target session

## Additional Resources

- [MCP Protocol Specification](https://modelcontextprotocol.io/)
- [Sliver Documentation](https://sliver.sh/)
- [AI Agent Integration Examples](https://github.com/BishopFox/sliver/tree/master/examples/ai-agents)
