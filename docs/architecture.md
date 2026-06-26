# Baize 架构

## 定位

Baize 是 AI Agent 引擎，不绑定特定前端。通过本地 HTTP API 向客户端暴露能力。

```
Cangjie (IDE插件)  │  Gewu (浏览器插件)  │  未来客户端
       │                    │                   │
       └────────────────────┼───────────────────┘
                            │
                   Baize API (HTTP+SSE)
                   localhost:9779/api/v1
                            │
              ┌─────────────┴─────────────┐
              │     Agent 引擎 (Go)        │
              │  ReAct · Supervisor        │
              │  Tools · MCP · Session     │
              │  Permission · Memory       │
              └───────────────────────────┘
```

## 核心包

| 包 | 职责 |
|------|------|
| `agent/` | Agent 抽象：ReAct / Supervisor / RAG，Graphflow 图节点，LLM 接口 |
| `tool/` | 工具接口 + 注册表。`builtin/` 6 个内置工具，`mcp/` MCP 客户端 |
| `server/` | HTTP+SSE API 服务，结构化路由，中间件 |
| `api/` | API 协议类型，`ToolProvider` 和 `MemoryProvider` 接口 |
| `sdk/` | Go HTTP 客户端，封装全部 API |
| `session/` | SQLite 会话持久化 |
| `permission/` | 策略引擎，已接入 `ToolNode.Run()` 执行前检查，不可绕过 |
| `memory/` | 文件式长期记忆 |
| `context/` | 项目文件分析（语言/构建工具/代码统计）|
| `tui/` | Bubble Tea 全屏终端 |
| `cmd/baize/` | CLI 入口，串起全部组件 |

## 关键依赖

```
baize → weave    (图执行引擎)
      → llmgate  (LLM 多模型网关)
      → mcp-go   (MCP 协议)
      → sqlite   (会话存储)
      → bubbletea (TUI)
```

## 安全模型

权限检查在 `ToolNode.executeToolCall()` 中，**先于 `tool.Execute()` 执行**。LLM 无法绕过。

```
LLM 产生 tool_call
  → ToolNode.executeToolCall()
    → PermissionChecker.CheckPermission(toolName, args)
      → "allow" → 执行
      → "deny"  → 拒绝 + 返回错误信息
      → "ask"   → 要求用户确认
```
