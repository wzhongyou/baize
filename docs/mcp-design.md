# Baize MCP 系统技术设计

> 关联文档：[设计 V1.1](design-v1.1.md)

---

## 概述

Baize 的 MCP 支持双向：

| 方向 | 组件 | 说明 |
|------|------|------|
| **Client**（消费者） | `core/tool/mcp/client.go` + `manager.go` | 连接外部 MCP server，将其工具注册进 ToolRegistry |
| **Server**（提供者） | `core/tool/mcp/server.go` | 将 Baize 自身工具暴露为 MCP server，供外部 MCP client 调用 |

---

## MCP Client

### 工作原理

```
外部 MCP Server 进程
      ↑ stdio (JSON-RPC)
ClientAdapter.Connect()
      ↓
mcpToolAdapter × N   →   ToolRegistry   →   agent LLM 可调用
```

1. `NewClientAdapter(command, args...)` 通过 stdio 启动外部进程
2. `Connect(ctx)` 发送 `Initialize`，拉取 `ListTools`
3. 每个 MCP 工具包装为 `mcpToolAdapter`（实现 `tool.Tool`），注册到 `ToolRegistry`
4. Agent 调用工具时，`Execute` 转发 `CallTool` 请求，返回文本内容

```
core/tool/mcp/client.go     ClientAdapter、mcpToolAdapter
core/tool/mcp/manager.go    Manager：多 server 生命周期管理
```

### Manager

```go
m := mcp.NewManager()
m.AddServer(ctx, "name", "npx", "-y", "@pkg/mcp-server")  // 启动并连接
tools := m.Tools()   // 所有 server 的工具聚合
m.Close()            // 关闭所有进程
```

`Manager` 是进程级单例，被 `skill.Manager` 持有（Skill 附带的 MCP server 通过此管理）。独立使用时可直接在 `buildToolRegistry` 中实例化。

### 工具输出

MCP 工具返回的文本结果经过 `truncateToolOutput`（20000 字符上限）后注入 MessageState，与内置工具完全一致。

支持富内容：MCP 工具可返回 `__baize_blocks` JSON 信封，`streamHook` 检测后转为 `ContentBlock` 推送给客户端。详见[会话协议](design-v1.1.md)。

---

## MCP Server（Baize 作为提供者）

```
core/tool/mcp/server.go     BaizeMCPServer（当前为 stub）
```

`BaizeMCPServer` 将 Baize 的 `ToolRegistry` 暴露为标准 MCP server，外部 MCP client（其他 AI 工具、IDE 插件）可通过 MCP 协议调用 Baize 的所有工具。

当前状态：结构已定义，`Serve(ctx)` 为 stub。完整实现（stdio JSON-RPC、Initialize/ListTools/CallTool 处理）为 P2。

---

## 配置方式

### 独立 MCP server（不通过 Skill）

在 `.baize/settings.toml` 中声明（待实现）：

```toml
[[mcp.servers]]
name = "filesystem"
command = "npx"
args = ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
```

### 通过 Skill 附带

在 Skill 目录的 `mcp.json` 中声明，随 Skill 生命周期启动和关闭。详见 [skill-design.md](skill-design.md)。

---

## 实现优先级

| 优先级 | 内容 |
|--------|------|
| P1 | MCP Client：ClientAdapter、Manager、工具注册 ✅ |
| P1 | Skill 附带 MCP server 自动启动 ✅ |
| P2 | `settings.toml` 独立 MCP server 配置 |
| P2 | BaizeMCPServer 完整 stdio 实现 |
| P3 | MCP over SSE（远程 MCP server） |
