# Baize Agent 引擎框架设计

> 关联文档：[设计 V1.1](design-v1.1.md) | 沙箱见 [sandbox-design.md](sandbox-design.md) | 多智能体见 [multi-agent-design.md](multi-agent-design.md)

---

## 概述

Agent 引擎是 Baize 的核心执行框架，基于 `github.com/wzhongyou/weave` 图引擎构建，将 LLM 推理、工具执行、状态管理组织为 DAG（有向无环图）节点流水线。

---

## 核心架构

```
MessageState（共享状态）
      │
      ▼
  Graph[*MessageState]（weave 图引擎）
      │
      ├── LLMNode      调用 LLM，追加 assistant 消息
      ├── ToolNode     执行工具调用，追加 tool_result 消息
      └── (条件边)     HasPendingToolCalls → 路由到 ToolNode
```

**weave 图引擎**（`github.com/wzhongyou/weave/graph`）负责：节点调度、条件路由、最大迭代控制、Hook 回调、并行节点执行。

---

## 核心数据结构

### MessageState

```go
type MessageState struct {
    Messages        []Message      // 对话历史（system/user/assistant/tool）
    Context         map[string]any // 跨节点传递的任意上下文
    CurrentAgent    string         // Supervisor 模式下当前执行的子 agent
    CompletedAgents []string
    StepCount       int
    MaxSteps        int
    TotalTokens     int
    Metadata        map[string]any
}
```

### Message

```go
type Message struct {
    Role             Role           // user | assistant | system | tool
    Content          string
    Images           []string       // base64 图片，仅 user 消息
    ReasoningContent string         // 思考模式推理内容
    ToolCalls        []ToolCall
    ToolCallID       string         // 匹配 ToolCall.ID，仅 tool 消息
    Timestamp        time.Time
}
```

---

## 节点类型

### LLMNode

调用 `LLMModel.ChatStream`，接收流式 delta 推送给 `OnChunk`，完成后把 assistant 消息追加到 `MessageState.Messages`。

配置项：

| 字段 | 说明 |
|------|------|
| `Model` | LLMModel 实现（llmgate Adapter） |
| `SystemPrompt` | 系统提示词 |
| `Tools` | 注册的工具列表（转为 ToolDef 传给 LLM） |
| `Stream` | 是否流式（默认 true） |
| `OnChunk` | delta 回调（推 SSE 事件） |
| `TodoManager` | 每次调用前注入 todo.md |
| `StructuredOutput` | JSON Schema 约束输出格式 |

### ToolNode

读取最后一条 assistant 消息的 `ToolCalls`，逐个（或并行）执行，结果追加为 `tool` 消息。

```go
// 并行执行（parallel=true）
for _, tc := range toolCalls {
    go func(tc ToolCall) { ch <- executeToolCall(ctx, tc) }(tc)
}
```

权限检查在工具执行前强制运行，不可绕过。

### supervisorRouteNode（Supervisor 模式）

读取 supervisor LLM 的 `route(agent)` 工具调用，构建子 agent 图并执行。

**关键设计**：子 agent 在独立 `MessageState` 中运行，完成后只把最终 assistant 消息作为 `tool_result` 返回，不污染主 context。

---

## Agent 类型

### ReActAgent

```
llm ──(HasPendingToolCalls)──→ tool ──→ llm（循环）
```

主力 agent，用于大多数编程任务。`MaxSteps` 防无限循环。

### SupervisorAgent

```
supervisor_llm ──(route 工具调用)──→ route ──→ collect ──→ supervisor_llm
```

路由任务到子 agent。内置子 agent 角色（计划中）：

| 角色 | 工具 | 模型 | 用途 |
|------|------|------|------|
| explore | file_read, grep, glob（只读） | 轻量模型 | 代码搜索理解 |
| plan | 只读 | 主模型（思考模式） | 任务分解 |
| edit | file_*, shell | 主模型 | 代码变更 |
| test | shell, file_read | 主模型 | 验证 |

### RAGAgent

```
retrieve ──→ llm
```

向量检索 + 生成，用于知识库问答场景。

---

## Hook 系统

```go
type Hook interface {
    OnGraphStart(ctx, name, state)
    OnGraphEnd(ctx, name, state, err)
    OnNodeStart(ctx, name, state)
    OnNodeEnd(ctx, name, state, err, duration)
    OnRetry(ctx, name, attempt, err)
}
```

`streamHook` 实现此接口，在 `OnNodeEnd` 时把 `tool_call` / `tool_result` 事件推送到 SSE。

---

## 执行流程（完整）

```
POST /chat
  │
  ├─ 加载历史（SQLite → MessageState.Messages）
  ├─ 注入系统提示 + BAIZE.md + skill 索引
  │
  ▼
agentRunner.RunStream(AgentRunRequest)
  │
  ├─ 构建 ReActAgent.BuildGraph()
  ├─ 注册 streamHook
  │
  ▼
engine.Run(ctx, state)
  │
  ├─ [LLMNode] → OnChunk → SSE thought/answer delta
  ├─ [ToolNode] → OnNodeEnd → SSE tool_call / tool_result
  │   └─ PermissionChecker → AskFunc（TUI/API 确认）
  │   └─ carrel Sandbox.Run（OS 级隔离）
  └─ done → SSE done event
```

---

## 用户确认流程

### 三档决策

| 决策 | 含义 | 持久化 |
|------|------|--------|
| `allow_once` | 本次放行 | 不持久 |
| `allow_session` | 本 session 不再问同类命令 | session 内存（carrel MemoryApprover） |
| `deny` | 拒绝，返回 Permission denied | 不持久 |

### 触发条件（三档模式）

```
suggest    所有写操作都 ask
auto-edit  file_* 自动执行，shell 每次 ask
full-auto  按 .baize/settings.toml glob 规则
```

### API 层确认协议

```
SSE: {"type":"permission_request","tool":"shell","args":{"cmd":"rm -rf dist/"},"reason":"危险命令"}
     ↓ 客户端响应
POST /api/v1/sessions/{id}/confirm
     {"request_id":"req-xxx","decision":"allow_once"}
```

`AskFunc` 阻塞等待，超时 30s 默认 deny。

### 权限规则持久化

```toml
# .baize/settings.toml
[permissions]
allow = ["shell:go test *", "shell:go build *", "file_edit:src/**"]
deny  = ["shell:rm -rf *", "shell:curl * | *"]
ask   = ["shell:*", "file_edit:*.toml"]
```

规则匹配顺序：`deny > allow > ask > 默认`。

---

## 实现优先级

| 优先级 | 内容 | 状态 |
|--------|------|------|
| P0 | ReActAgent + LLMNode + ToolNode + 并行工具 | ✅ |
| P0 | streamHook SSE 推送 | ✅ |
| P0 | PermissionChecker 三档模式 | ✅ |
| P1 | SupervisorAgent 上下文隔离修复 | 待实现 |
| P1 | explore 内置子 agent（只读+轻模型） | 待实现 |
| P1 | API 层 permission_request + confirm 端点 | 待实现 |
| P1 | .baize/settings.toml glob 规则 | 待实现 |
| P2 | 阶段化执行 + 验证门 | 见 multi-agent-design.md |

---

## 错误恢复与自愈

### 工具调用失败重试策略

工具返回 `error:` 前缀结果时，LLM 当前靠自然语言推理决定下一步，没有结构化保障。

设计三级重试策略（计划中）：

| 失败类型 | 策略 |
|---------|------|
| 文件不存在 | 自动触发 `grep`/`glob` 搜索正确路径，重试 |
| 权限拒绝 | 停止重试，推送 `permission_request` 给用户 |
| 命令超时 | 在 tool_result 追加提示"命令超时，建议拆分或缩小范围" |
| LLM 幻觉（路径/函数不存在）| 追加系统消息"请先用 glob/grep 确认路径存在" |

### 自我反思节点（计划中）

参考 Claude Code 的"think before acting"模式，在高风险工具调用前插入反思步骤：

```
[tool_call: file_edit]
    ↓
[reflection node] 检查：old_string 是否唯一？影响范围是否符合预期？
    ↓ 确认
[execute]
```

反思节点使用 `StructuredOutput` 约束 LLM 输出结构化判断，不依赖 LLM 自然语言决策。

### 幻觉检测

见 [eval-design.md](eval-design.md) 幻觉检测章节。检测到幻觉时自动注入纠正提示，不中断执行。
