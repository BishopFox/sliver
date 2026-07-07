# Sliver MCP Server 新增工具

本次更新为 Sliver C2 的 MCP Server 添加了 5 个新工具，扩展了远程目标系统的监控和操作能力。

## 新增工具列表

### 1. list_processes - 列出进程
**功能**: 列出目标主机上运行的所有进程

**参数**:
- `session_id` (可选): Session ID
- `beacon_id` (可选): Beacon ID
- `full_info` (可选): 是否获取完整进程信息
- `wait` (可选): 是否等待 Beacon 任务完成
- `timeout_seconds` (可选): 超时时间（秒）

**返回**: 进程列表，包含 PID、PPID、可执行文件路径、所有者、架构、命令行等信息

**RPC 方法**: `Ps`

---

### 2. kill_process - 终止进程
**功能**: 终止目标主机上的指定进程

**参数**:
- `session_id` (可选): Session ID
- `beacon_id` (可选): Beacon ID
- `pid` (必需): 要终止的进程 ID
- `force` (可选): 是否强制终止
- `wait` (可选): 是否等待 Beacon 任务完成
- `timeout_seconds` (可选): 超时时间（秒）

**返回**: 终止结果，包含 PID 和成功状态

**RPC 方法**: `Terminate`

---

### 3. list_credentials - 列出凭证
**功能**: 列出已收集的凭证信息

**参数**: 无（全局查询）

**返回**: 凭证列表，包含：
- ID
- 用户名
- 明文密码（如果有）
- 哈希值
- 哈希类型（支持多种格式：MD5, SHA1, SHA256, NTLM, Kerberos 等）
- 是否已破解
- 来源主机 UUID
- 集合名称

**RPC 方法**: `Creds`

---

### 4. network_interfaces - 网络接口信息
**功能**: 获取目标主机的网络接口配置

**参数**:
- `session_id` (可选): Session ID
- `beacon_id` (可选): Beacon ID
- `wait` (可选): 是否等待 Beacon 任务完成
- `timeout_seconds` (可选): 超时时间（秒）

**返回**: 网络接口列表，包含：
- 索引
- 接口名称
- MAC 地址
- IP 地址列表

**RPC 方法**: `Ifconfig`

---

### 5. netstat - 网络连接状态
**功能**: 获取目标主机的网络连接状态（类似 netstat/ss 命令）

**参数**:
- `session_id` (可选): Session ID
- `beacon_id` (可选): Beacon ID
- `tcp` (可选): 是否显示 TCP 连接
- `udp` (可选): 是否显示 UDP 连接
- `ip4` (可选): 是否显示 IPv4 连接
- `ip6` (可选): 是否显示 IPv6 连接
- `listening` (可选): 是否只显示监听端口
- `wait` (可选): 是否等待 Beacon 任务完成
- `timeout_seconds` (可选): 超时时间（秒）

**返回**: 网络连接列表，包含：
- 协议类型
- 本地地址（IP 和端口）
- 远程地址（IP 和端口）
- 连接状态
- UID
- 关联进程信息（PID、可执行文件、所有者）

**RPC 方法**: `Netstat`

---

## 实现细节

### 文件结构
- `processes.go`: 实现 list_processes 和 kill_process 工具
- `credentials.go`: 实现 list_credentials 工具
- `network.go`: 实现 network_interfaces 和 netstat 工具
- `server.go`: 更新工具注册逻辑

### 特性
1. **完整的错误处理**: 所有工具都包含 RPC 错误、参数验证和超时处理
2. **Beacon 异步支持**: 支持异步 Beacon 任务，可选择等待或立即返回任务 ID
3. **结构化 JSON 输出**: 使用 `NewToolResultStructuredOnly()` 返回格式化的 JSON 数据
4. **日志记录**: 所有工具调用都会记录到 MCP 日志系统
5. **哈希类型映射**: credentials.go 包含完整的 HashType 枚举到字符串的映射

### 代码风格
- 遵循现有 MCP 工具的代码风格
- 使用相同的参数验证和错误处理模式
- 保持一致的命名约定和注释风格
- 支持 Session 和 Beacon 两种模式

## 工具总数
现在 Sliver MCP Server 共有 **20 个工具**：
- 原有 15 个工具
- 新增 5 个工具

## 使用示例

```json
// 列出进程
{
  "tool": "list_processes",
  "arguments": {
    "session_id": "abc123",
    "full_info": true
  }
}

// 终止进程
{
  "tool": "kill_process",
  "arguments": {
    "session_id": "abc123",
    "pid": 1234,
    "force": true
  }
}

// 列出凭证
{
  "tool": "list_credentials",
  "arguments": {}
}

// 获取网络接口
{
  "tool": "network_interfaces",
  "arguments": {
    "session_id": "abc123"
  }
}

// 获取网络连接
{
  "tool": "netstat",
  "arguments": {
    "session_id": "abc123",
    "tcp": true,
    "listening": true
  }
}
```
