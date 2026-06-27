# Baize Server 运行时设计

> 关联文档：[设计 V1.1](design-v1.1.md) | 会话存储见 [session-design.md](session-design.md)

---

## 概述

`server/` 包是 Baize 的 HTTP + SSE 运行时，职责：

1. 接收 HTTP 请求，路由到对应 handler
2. **请求调度**：同一 session 串行化，防止并发冲突
3. **SSE 流式输出**：agent 事件实时推送给客户端
4. 管理 AgentRunner 生命周期

**传输层**：Go 标准库 `net/http`，无第三方框架。流式输出用 SSE（`text/event-stream` + `http.Flusher`），单向推送场景无需 WebSocket。

---

## API 全量清单

所有对外暴露的 HTTP 接口。字段细节见各子文档链接。

### 基础

| 方法 | 路径 | 说明 | 详细设计 |
|------|------|------|---------|
| `GET` | `/api/v1/health` | 健康检查，返回 `{"status":"ok","version":"v1"}` | — |

### 会话

| 方法 | 路径 | 说明 | 详细设计 |
|------|------|------|---------|
| `POST` | `/api/v1/chat` | 发起/继续会话，SSE 流式响应 | [chat-protocol-design.md](chat-protocol-design.md) |
| `GET` | `/api/v1/sessions` | 列出会话（最近 100 条，按 updated_at 倒序） | [session-design.md](session-design.md) |
| `GET` | `/api/v1/sessions/:id` | 会话详情 + 消息列表 | [session-design.md](session-design.md) |
| `DELETE` | `/api/v1/sessions/:id` | 删除会话及所有消息 | [session-design.md](session-design.md) |
| `GET` | `/api/v1/sessions/:id/stream` | 订阅 pending 任务的 SSE 流（计划中） | [server-design.md](#响应策略) |
| `POST` | `/api/v1/sessions/:id/confirm` | 响应 permission_request，返回用户确认决策 | [agent-engine-design.md](agent-engine-design.md) |

### 工具

| 方法 | 路径 | 说明 | 详细设计 |
|------|------|------|---------|
| `GET` | `/api/v1/tools` | 列出已注册工具（name/description/parameters/source） | [tool-design.md](tool-design.md) |
| `POST` | `/api/v1/tools/call` | 直接调用单个工具（调试用） | [tool-design.md](tool-design.md) |

### 记忆

| 方法 | 路径 | 说明 | 详细设计 |
|------|------|------|---------|
| `POST` | `/api/v1/memory/search` | 关键词搜索记忆，返回 top-K 结果 | [memory-design.md](memory-design.md) |
| `POST` | `/api/v1/memory/save` | 保存记忆条目 | [memory-design.md](memory-design.md) |

### 通用响应格式

```json
{
  "code": 0,
  "data": {...},
  "message": "",
  "request_id": "req-xxx"
}
```

`code=0` 为成功，非零为错误（错误码见 [chat-protocol-design.md](chat-protocol-design.md)）。`/api/v1/chat` 例外，直接返回 SSE 流。

> **CLI 不走 HTTP**：CLI 直接调用 `session.Store`、`ToolRegistry` 等内部接口，操作同一个 `baize.db`。HTTP API 供 VSCode 插件、Web、第三方应用集成使用。详见 [session-design.md](session-design.md)。

### Middleware 链

`RequestID → CORS → Logging → RateLimit（计划中）`

---

## 请求调度与并发控制（计划中）

### 问题

同一 session 并发请求导致：
- 多个 agent 同时操作文件系统/session 状态
- SSE 并发写 `ResponseWriter`（race condition）

### 任务状态机

```
pending → running → done / cancelled
```

### 新消息处理规则

| session 当前状态 | 新消息处理方式 |
|-----------------|---------------|
| 空闲 | 直接进入 `running`，建立 SSE 流 |
| `pending`（已入队，LLM 尚未调用） | **合并**进 pending 任务的 context，不新建任务 |
| `running`（agent 执行中） | 新建任务排入 `pending`，返回 `202 Accepted` |

**pending 阶段可合并的原因**：任务尚未发给 LLM，用户追加的说明直接拼入 context，agent 执行时看到完整信息，避免两个割裂任务。

**pending 最多一个**：后续消息继续合并进已有 pending 任务，不无限积压。

### SessionDispatcher 结构

```go
// server/dispatcher.go
type SessionDispatcher struct {
    mu      sync.Mutex
    queues  map[string]*SessionQueue  // key = session_id
}

type SessionQueue struct {
    mu      sync.Mutex
    running *Task
    pending *Task
    cancel  context.CancelFunc
}

type Task struct {
    ID       string
    Messages []string   // 合并后的用户消息列表
    Images   [][]string // 对应每条消息的图片
    ResultCh chan StreamEvent
}
```

### 响应策略

- **running 任务**：HTTP 连接保持，直接推送 SSE 事件流至 done
- **pending 任务**：返回 `202 Accepted`
  ```json
  {"task_id": "task-xxx", "session_id": "sess-yyy", "position": 1}
  ```
  客户端通过 `GET /api/v1/sessions/{id}/stream` 提前建立 SSE 连接，等任务进入 running 后开始推送

### 当前状态

尚未实现。现有 `handleChat` 无 session 级串行化，高并发下存在 race condition 风险。实现优先级：P1。

---

## SSE 写入串行化（计划中）

并行工具执行时，多个 goroutine 同时调用 `onEvent`，写 `ResponseWriter` 不安全。

解决方案：buffered channel + 单写 goroutine：

```go
evCh := make(chan protocol.ChatEvent, 32)
go func() {
    for ev := range evCh {
        sendSSE(ev)
    }
}()
onEvent := func(ev StreamEvent) {
    evCh <- toAPIEvent(ev)
}
```

当前状态：尚未实现，并行工具场景存在并发写风险。

---

## AgentRunner 接口

```go
type AgentRunner interface {
    Run(ctx context.Context, req AgentRunRequest) (*AgentRunResult, error)
    RunStream(ctx context.Context, req AgentRunRequest, onEvent func(StreamEvent))
}
```

`cli/main.go` 中的 `agentRunner` 实现此接口，将 HTTP 层与 agent core 解耦。`StreamEvent` 内部类型在 handler 中映射为 `protocol.ChatEvent` 后推送 SSE。

---

## 实现优先级

| 优先级 | 内容 |
|--------|------|
| P0 | HTTP + SSE handler、AgentRunner 接口 ✅ |
| P1 | SessionDispatcher：pending 合并 + running 串行化 |
| P1 | SSE 写入串行化（evCh goroutine） |
| P1 | `POST /sessions/:id/confirm` 确认端点 |
| P2 | `GET /sessions/:id/stream` 订阅端点 |
| P2 | 限流（RateLimit middleware） |


> 关联文档：[设计 V1.1](design-v1.1.md) | 会话存储见 [session-design.md](session-design.md)

---

## 概述

`server/` 包是 Baize 的 HTTP + SSE 运行时，职责：

1. 接收 HTTP 请求，路由到对应 handler
2. **请求调度**：同一 session 串行化，防止并发冲突
3. **SSE 流式输出**：agent 事件实时推送给客户端
4. 管理 AgentRunner 生命周期

---

## HTTP 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/health` | 健康检查 |
| `POST` | `/api/v1/chat` | 发起/继续会话，SSE 流式响应 |
| `GET` | `/api/v1/sessions` | 列出会话 |
| `GET` | `/api/v1/sessions/:id` | 会话详情 + 消息 |
| `DELETE` | `/api/v1/sessions/:id` | 删除会话 |
| `GET` | `/api/v1/tools` | 列出已注册工具 |
| `POST` | `/api/v1/memory/search` | 记忆搜索 |
| `POST` | `/api/v1/memory/save` | 保存记忆 |

Middleware 链：`RequestID → CORS → Logging → RateLimit`

---

## 请求调度与并发控制（计划中）

### 问题

同一 session 并发请求导致：
- 多个 agent 同时操作文件系统/session 状态
- SSE 并发写 `ResponseWriter`（race condition）

### 任务状态机

```
pending → running → done / cancelled
```

### 新消息处理规则

| session 当前状态 | 新消息处理方式 |
|-----------------|---------------|
| 空闲 | 直接进入 `running`，建立 SSE 流 |
| `pending`（已入队，LLM 尚未调用） | **合并**进 pending 任务的 context，不新建任务 |
| `running`（agent 执行中） | 新建任务排入 `pending`，返回 `202 Accepted` |

**pending 阶段可合并的原因**：任务尚未发给 LLM，用户追加的说明直接拼入 context，agent 执行时看到完整信息，避免两个割裂任务。

**pending 最多一个**：后续消息继续合并进已有 pending 任务，不无限积压。

### SessionDispatcher 结构

```go
// server/dispatcher.go
type SessionDispatcher struct {
    mu      sync.Mutex
    queues  map[string]*SessionQueue  // key = session_id
}

type SessionQueue struct {
    mu      sync.Mutex
    running *Task
    pending *Task
    cancel  context.CancelFunc
}

type Task struct {
    ID       string
    Messages []string   // 合并后的用户消息列表
    Images   [][]string // 对应每条消息的图片
    ResultCh chan StreamEvent
}
```

### 响应策略

- **running 任务**：HTTP 连接保持，直接推送 SSE 事件流至 done
- **pending 任务**：返回 `202 Accepted`
  ```json
  {"task_id": "task-xxx", "session_id": "sess-yyy", "position": 1}
  ```
  客户端通过 `GET /api/v1/sessions/{id}/stream` 提前建立 SSE 连接，等任务进入 running 后开始推送

### 当前状态

尚未实现。现有 `handleChat` 无 session 级串行化，高并发下存在 race condition 风险。实现优先级：P1。

---

## SSE 写入串行化（计划中）

并行工具执行时，多个 goroutine 同时调用 `onEvent`，写 `ResponseWriter` 不安全。

解决方案：buffered channel + 单写 goroutine：

```go
evCh := make(chan protocol.ChatEvent, 32)
go func() {
    for ev := range evCh {
        sendSSE(ev)
    }
}()
onEvent := func(ev StreamEvent) {
    evCh <- toAPIEvent(ev)
}
```

当前状态：尚未实现，并行工具场景存在并发写风险。

---

## AgentRunner 接口

```go
type AgentRunner interface {
    Run(ctx context.Context, req AgentRunRequest) (*AgentRunResult, error)
    RunStream(ctx context.Context, req AgentRunRequest, onEvent func(StreamEvent))
}
```

`cli/main.go` 中的 `agentRunner` 实现此接口，将 HTTP 层与 agent core 解耦。`StreamEvent` 内部类型在 handler 中映射为 `protocol.ChatEvent` 后推送 SSE。

---

## 实现优先级

| 优先级 | 内容 |
|--------|------|
| P0 | HTTP + SSE handler、AgentRunner 接口 ✅ |
| P1 | SessionDispatcher：pending 合并 + running 串行化 |
| P1 | SSE 写入串行化（evCh goroutine） |
| P2 | `GET /sessions/{id}/stream` 订阅端点 |
| P2 | 限流（RateLimit middleware） |
