# Baize 架构

## 设计原则

| 原则 | 说明 |
|------|------|
| **模型无关** | 通过 llmgate 接入 20+ 提供商，不绑定单一模型 |
| **图编排** | Agent 逻辑以 weave 有向图运行，节点、边、条件清晰可测 |
| **不可绕过权限** | 权限检查嵌入 ToolNode，执行前强制判定，LLM 无法越权 |
| **单二进制** | Go 编译，零依赖部署，一个二进制覆盖 CLI/TUI/Server |

---

## 系统架构总览

```
                          ┌──────────────────────────────────────────────────────────────────────┐
                          │                            客户端层                                   │
                          │  ┌──────────┐  ┌──────────┐  ┌───────────┐  ┌──────────────────┐  │
                          │  │   TUI    │  │  VSCode  │  │ JetBrains   │  │ JetBrains     │  │
                          │  │ BubbleTea│  │ Extension│  │  Plugin│  │  Plugin     │  │
                          │  └────┬─────┘  └────┬─────┘  └─────┬─────┘  └────────┬─────────┘  │
                          └───────┼─────────────┼──────────────┼─────────────┼─────────────┘
                                  │             │              │             │
                                  └─────────────┴──────────────┴──────┬──────┘
                                                                      │
┌─────────────────────────────────────────────────────────────────────┼──────────────────────────┐
│                              API 层                                 │                          │
│  ┌─────────────────────────────────────────────────────────────────┴────────────────────────┐ │
│  │                         HTTP + SSE  (localhost:9779)                                      │ │
│  │  GET /health   POST /chat   POST /tools/*   GET /sessions/*   POST /memory/*            │ │
│  └─────────────────────────────────────────────────────────────────┬────────────────────────┘ │
│                                                                    │                          │
│  Middleware:  RequestID  ·  CORS  ·  Logging                       │                          │
└────────────────────────────────────────────────────────────────────┼──────────────────────────┘
                                                                     │
                              AgentRunner { Run(), RunStream() }
                                                                     │
┌────────────────────────────────────────────────────────────────────┼──────────────────────────┐
│                          Agent 引擎                                │                          │
│                                                                    │                          │
│  ┌─────────────────────────────────────────────────────────────────┴────────────────────────┐ │
│  │                          weave Graph Engine                                              │ │
│  │                                                                                          │ │
│  │  ┌──────────────────────┐    HasPendingToolCalls?    ┌──────────────────────┐           │ │
│  │  │      LLMNode         │ ─────────────────────────→ │      ToolNode        │           │ │
│  │  │                      │ ←───────────────────────── │                      │           │ │
│  │  │  · buildMessages     │                            │  · executeToolCall   │           │ │
│  │  │  · Chat / ChatStream │                            │    ├─ SafeTool 元数据 │           │ │
│  │  │  · OnChunk 流式回调   │                            │    ├─ PermissionCheck │           │ │
│  │  │  · StructuredOutput  │                            │    ├─ AskFunc(I/O)    │           │ │
│  │  └──────────────────────┘                            │    └─ tool.Execute()  │           │ │
│  │                                                      └──────────────────────┘           │ │
│  │  ┌──────────────────────┐    ┌──────────────────────┐    ┌──────────────────────┐       │ │
│  │  │   SupervisorAgent    │    │      RAGAgent        │    │     Hooks            │       │ │
│  │  │   多智能体路由调度     │    │   VectorRetrieveNode │    │  OnGraphStart/End    │       │ │
│  │  │   route → sub-agent  │    │   Embed → Search     │    │  OnNodeStart/End     │       │ │
│  │  └──────────────────────┘    └──────────────────────┘    └──────────────────────┘       │ │
│  └────────────────────────────────────────────────────┬─────────────────────────────────────┘ │
│                                                       │                                       │
├───────────────────────────────────────────────────────┼───────────────────────────────────────┤
│                        状态 & 数据层                   │                                       │
│  ┌────────────────────────────┐  ┌────────────────────┴──────────────┐  ┌──────────────────┐ │
│  │       MessageState         │  │      Session (SQLite)             │  │  Context 分析    │ │
│  │                            │  │                                   │  │                  │ │
│  │  · Messages []Message      │  │  sessions 表: id, title, model,  │  │  · 语言检测      │ │
│  │  · StepCount / MaxSteps    │  │    workspace, status, tokens     │  │  · 框架识别      │ │
│  │  · TotalTokens             │  │  messages 表: role, content,     │  │  · 构建工具      │ │
│  │  · Context map (跨节点)     │  │    tool_calls(JSON), session_id  │  │  · 目录树        │ │
│  │  · CurrentAgent (路由)     │  │  WAL 模式 · 外键级联 · RWMutex  │  │  · 代码统计      │ │
│  └────────────────────────────┘  └─────────────────────────────────┘  └──────────────────┘ │
│  ┌────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                               Memory 记忆系统                                           │ │
│  │  ┌──────────────────────┐  ┌──────────────────────┐  ┌──────────────────────────────┐  │ │
│  │  │  ShortTermMemory     │  │  LongTermMemory      │  │  MarkdownStore               │  │ │
│  │  │  环形缓冲 · 保护system│  │  Embed → VectorStore │  │  .md 文件 · YAML frontmatter │  │ │
│  │  │  prompt · 逐出旧消息  │  │  语义检索 · topK     │  │  关键词匹配 · 文件系统存储   │  │ │
│  │  └──────────────────────┘  └──────────────────────┘  └──────────────────────────────┘  │ │
│  └────────────────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    工具 & 扩展层                                              │
│                                                                                             │
│  ┌───────────────────────────────────── ToolRegistry ─────────────────────────────────────┐ │
│  │                              tool.Tool 接口  +  tool.SafeTool 接口                      │ │
│  └──────────────────────────┬─────────────────────────────────────────────────────────────┘ │
│                             │                                                               │
│  ┌──────────────────────────┼──────────────────────────┬──────────────────────────────────┐ │
│  │     6 个内置工具          │      MCP Manager          │       Permission Engine         │ │
│  │                          │                            │                                  │ │
│  │  file   read/write/edit  │  AddServer(name,cmd,args) │  PolicyEngine                    │ │
│  │         list/search      │  RemoveServer(name)       │  ├─ PolicyRule[] (优先级匹配)     │ │
│  │  shell  命令执行 120s超时 │  Tools() []Tool           │  ├─ Learn(ScopeAlways)           │ │
│  │         危险命令屏蔽      │  Close()                  │  ├─ DefaultPolicy()              │ │
│  │  git    status/diff/log  │                            │  └─ AsAgentChecker(reg)          │ │
│  │         add/commit/br    │  ┌────────────────────┐   │                                  │ │
│  │  web    search / fetch   │  │  MCP Server A       │   │  PermissionChecker 接口          │ │
│  │  calc   表达式求值        │  │  (filesystem)       │   │  ├─ allow → Execute             │ │
│  │                          │  ├────────────────────┤   │  ├─ deny  → 返回拒绝信息          │ │
│  │                          │  │  MCP Server B       │   │  └─ ask   → AskFunc(用户确认)    │ │
│  │                          │  │  (database)         │   │                                  │ │
│  │                          │  └────────────────────┘   │  AuditLogger                     │ │
│  │                          │    stdio · JSON-RPC       │  决策记录 · 时间戳 · 追溯         │ │
│  └──────────────────────────┴──────────────────────────┴──────────────────────────────────┘ │
│                                                                                             │
│  ┌──────────────────────────────────────────────────────────────────────────────────────┐   │
│  │                              Carrel 沙箱 (规划)                                       │   │
│  │  ToolNode.executeToolCall() → cgroup/namespace 隔离 → 文件系统/网络策略 → 资源限制     │   │
│  └──────────────────────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    基础设施层                                                  │
│                                                                                             │
│  ┌────────────────────────────────────┐   ┌──────────────────────────────────────────────┐  │
│  │            llmgate 网关             │   │               外部依赖                        │  │
│  │                                    │   │                                              │  │
│  │  策略路由 · 降级 · 重试             │   │  SQLite (modernc.org/sqlite, 纯 Go)          │  │
│  │                                    │   │  Bubble Tea (TUI 框架)                       │  │
│  │  ┌────────┐ ┌────────┐ ┌────────┐ │   │  mcp-go (MCP 协议)                           │  │
│  │  │DeepSeek│ │OpenAI  │ │Anthropic│ │   │  weave (图执行引擎)                           │  │
│  │  ├────────┤ ├────────┤ ├────────┤ │   │                                              │  │
│  │  │ GLM    │ │Moonshot│ │Groq   │ │   │  单二进制部署:                                  │  │
│  │  ├────────┤ ├────────┤ ├────────┤ │   │  go build -o baize ./cli/               │  │
│  │  │ ...    │ │ ...    │ │ ...    │ │   │  Docker: alpine + ca-certs + git + curl       │  │
│  │  └────────┘ └────────┘ └────────┘ │   │                                              │  │
│  │      20+ 模型服务商                │   │                                              │  │
│  └────────────────────────────────────┘   └──────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 数据流（一次 Chat 请求的完整路径）

```
Client                     API Server                  Agent Engine               外部服务
  │                            │                            │                         │
  │  POST /api/v1/chat        │                            │                         │
  │  {message, session_id}    │                            │                         │
  │ ─────────────────────────→│                            │                         │
  │                            │                            │                         │
  │                            │  加载/创建 Session          │                         │
  │                            │  ├─ SQLite sessions 表     │                         │
  │                            │  └─ 恢复历史 Messages      │                         │
  │                            │                            │                         │
  │                            │  AgentRunner.RunStream()   │                         │
  │                            │ ──────────────────────────→│                         │
  │                            │                            │                         │
  │                            │                            │  weave Graph.Run()      │
  │                            │                            │  ┌─── llm 节点 ───┐     │
  │                            │                            │  │ buildMessages  │     │
  │                            │  SSE "thought"             │  │ ChatStream ────┼────→│ llmgate → LLM
  │  data: {"type":"thought"} │ ←────────────────────────── │  │ OnChunk 回调   │←────│ 流式 token
  │ ←─────────────────────────│                            │  └────────┬───────┘     │
  │                            │                            │           │              │
  │                            │                            │    HasToolCalls?        │
  │                            │                            │           │              │
  │                            │  SSE "tool_call"           │  ┌─── tool 节点 ───┐    │
  │  data: {"type":"tool_call"}│ ←────────────────────────── │  │ PermissionCheck │    │
  │ ←─────────────────────────│                            │  │ ├ allow/deny    │    │
  │                            │                            │  │ └ ask→AskFunc   │    │
  │                            │  SSE "permission_ask"      │  │    TUI弹窗等待   │    │
  │  data: {"type":"perm_ask"} │ ←────────────────────────── │  │ tool.Execute()  │    │
  │ ←─────────────────────────│                            │  └────────┬───────┘    │
  │  (用户确认 Y/N/A)          │                            │           │              │
  │ ─────────────────────────→│                            │           │              │
  │                            │                            │    继续循环...          │
  │                            │                            │           │              │
  │  SSE "answer"              │                            │  循环结束  │              │
  │ ←─────────────────────────│                            │           │              │
  │  SSE "done"                │                            │           │              │
  │ ←─────────────────────────│ 保存 assistant msg → SQLite │           │              │
```

---

## Agent 引擎

Agent 引擎基于 [Weave](https://github.com/wzhongyou/weave) 图执行框架，将 Agent 行为建模为有向图：**节点**执行操作，**边**控制流转，**状态**贯穿始终。

### 核心类型

```
ChatRequest ────────────────────── ChatResponse
    │                                    │
    │ Messages = [                       │ Content
    │   {Role: "system", ...},           │ ToolCalls[{Name, Args}]
    │   {Role: "user", ...},             │ ReasoningContent
    │   {Role: "assistant", ToolCalls}]   │ Usage
    │                                    │
    │ Tools = [{Name, Parameters}]       │
    └────────────────────────────────────┘
```

| 类型 | 包 | 职责 |
|------|-----|------|
| `LLMModel` | `agent/` | Chat / ChatStream 抽象，llmgate 是唯一实现 |
| `ChatRequest` | `agent/llm.go` | 入站：消息列表 + 工具定义 + 温度/Token 控制 |
| `ChatResponse` | `agent/llm.go` | 出站：文本内容 + 工具调用列表 + Token 用量 |
| `StreamChunk` | `agent/llm.go` | 流式增量：ChatResponse 字段的 partial 版本，通过 channel 推送 |
| `StreamEvent` | `server/` | 上层事件：thought / tool_call / tool_result / answer / done |

流式数据通路：

```
LLM Provider ──→ chan *StreamChunk ──→ LLMNode 攒批 ──→ StreamEvent ──→ SSE data:
```

### 图节点

每个节点都是 `func(ctx, *MessageState) (*MessageState, error)` 形态的纯函数，由 weave 引擎调度执行。

```
┌──────────────────────────────────────────────────────┐
│                     graph.Graph                      │
│                                                      │
│  ┌──────────┐    condition     ┌──────────┐         │
│  │ LLMNode  │ ───HasPending──→ │ ToolNode │         │
│  │          │   ToolCalls?     │          │         │
│  │ · 构建消息列表              │ · 提取 SafeTool 元数据  │
│  │ · 调用 LLM                 │ · PermissionChecker    │
│  │ · 流式输出                  │ · tool.Execute()       │
│  │ · 追加 assistant 消息       │ · 追加 tool 结果消息    │
│  └──────────┘                  └──────────┘         │
│       ↑                              │               │
│       └──────────────────────────────┘               │
│                   MaxSteps 兜底                       │
└──────────────────────────────────────────────────────┘
```

| 节点 | 职责 |
|------|------|
| `LLMNode` | 构建消息列表 → 注入系统提示词 + 工具定义 → 调用 LLM → 流式输出 → 结构化输出校验 → 追加 assistant 消息 |
| `ToolNode` | 遍历未执行的 ToolCall → 权限判定 → 执行工具 → 追加 tool 结果消息 |
| `VectorRetrieveNode` | Embed 最后一条消息 → 检索 VectorStore → 写入 `s.Context["retrieved_docs"]` |
| `supervisorRouteNode` | 解析 supervisor 的 route 调用 → 构建子 Agent 图 → 执行 → 合并状态 |

### AgentRunner：引擎与外界的边界

```go
// server/ 定义，cli/ 实现
type AgentRunner interface {
    Run(ctx, AgentRunRequest) (*AgentRunResult, error)            // 非流式
    RunStream(ctx, AgentRunRequest, onEvent func(StreamEvent))    // 流式
}
```

三种运行模式共享同一个 AgentRunner 实现：

```
./baize              TUI 模式    onEvent → Bubble Tea 消息 → 终端渲染
./baize "问题"       非交互模式   Run()   → 等待完成，打印最终答案
./baize server       API 模式    onEvent → SSE data: 帧 → 客户端消费
```

---

## CLI & TUI

### 模式

| 命令 | 模式 | 说明 |
|------|------|------|
| `baize` | TUI 交互 | 全屏 Bubble Tea 终端，流式渲染，权限弹窗 |
| `baize "query"` | 单次执行 | 非交互，打印最终答案 + 统计信息 |
| `baize server` | API 服务 | HTTP+SSE，端口 9779 |

### CLI 标志

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--config` | 自动搜索 | llmgate TOML 配置文件路径 |
| `--provider` | 自动路由 | 模型提供商（deepseek, openai, anthropic 等） |
| `--model` | 自动路由 | 模型 ID |
| `--workspace` | `.` | 工作区根目录 |
| `--max-steps` | `30` | Agent 最大执行步数 |
| `--verbose` | `false` | 详细输出 |
| `--no-tui` | `false` | 使用简单 REPL 替代 TUI |
| `--port` | `9779` | server 模式监听端口 |
| `--host` | `127.0.0.1` | server 模式绑定地址 |

配置文件查找顺序：`--config` > `conf/llmgate.toml` > `./llmgate.toml` > `~/.baize/config.toml`。Provider 和 Model 可从配置文件读取，无需 CLI 传参。

### TUI 架构

Bubble Tea Model-Update-View 模式，通过 channel 桥接 Agent 流式事件。

```
┌─────────────────────────────────────┐
│            TUI Model                │
│                                     │
│  ┌─────────────────────────────┐   │
│  │   Status Bar                 │   │
│  │   Baize v0.4 · session · model│   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │   Chat Viewport (可滚动)     │   │
│  │   · 用户消息 (蓝色)          │   │
│  │   · 助手回复 (灰色)          │   │
│  │   · 工具调用 (青色)          │   │
│  │   · 工具结果 (折叠/展开)     │   │
│  │   · 错误信息 (红色)          │   │
│  │   · 思考过程 (斜体)          │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │   Input Area (多行支持)      │   │
│  │   > 用户输入光标█             │   │
│  └─────────────────────────────┘   │
│                                     │
│  [模式: INPUT / THINKING / CONFIRM] │
└─────────────────────────────────────┘
```

### UI 模式

| 模式 | 说明 | 触发 |
|------|------|------|
| `modeInput` | 等待用户输入 | 启动、流式完成、确认完成 |
| `modeThinking` | Agent 执行中，流式渲染 | 用户提交后 |
| `modeConfirm` | 权限确认弹窗，阻塞 Agent | PermissionChecker 返回 "ask" |

### 键盘快捷键

| 快捷键 | 作用 |
|--------|------|
| `Enter` | 提交输入（单行）/ 插入换行（Shift+Enter） |
| `Shift+Enter` | 在输入中插入换行 |
| `Ctrl+A` / `Ctrl+E` | 跳到行首 / 行尾 |
| `Ctrl+U` | 清空当前输入 |
| `Backspace` | 删除光标前字符 |
| `←` / `→` | 光标左右移动 |
| `↑` / `↓` | 浏览历史命令 |
| `Ctrl+C` / `Esc` | 取消当前操作 / 退出 |
| `PageUp` / `PageDown` | 聊天视口翻页 |
| `Y` / `N` / `A` / `Esc` | 确认弹窗：允许 / 拒绝 / 始终允许 / 取消 |

粘贴（Ctrl+V / Cmd+V）原生支持，多字符粘贴不会被过滤。

### 流式事件映射

Agent 引擎的 `StreamEvent` → TUI 渲染：

| 事件类型 | TUI 行为 |
|----------|----------|
| `thought` | 追加到思考缓冲区（斜体灰色实时渲染） |
| `tool_call` | 插入工具调用气泡（工具名 + 参数摘要） |
| `tool_result` | 追加结果到对应工具气泡（可折叠长输出） |
| `answer` | 清空思考缓冲，插入最终回答气泡 |
| `done` | 结束流式，切换回 INPUT 模式 |
| `error` | 插入红色系统消息 |
| `permission_ask` | 切换到 CONFIRM 模式，弹窗等待用户决策 |

### 权限确认流程

```
ToolNode.executeToolCall()
  │
  └── permission check → "ask"
        │
        ▼
    AskFunc 阻塞等待
        │
        ▼
    TUI StreamEvent "permission_ask" → CONFIRM 弹窗
        │
    ┌─── Y ─── 允许本次
    ├─── N ─── 拒绝本次
    ├─── A ─── 始终允许（调用 PolicyEngine.Learn() 持久化）
    └─── Esc ─ 取消（同拒绝）
        │
        ▼
    AskFunc 返回结果 → ToolNode 执行或拒绝
```

### 斜杠命令

| 命令 | 说明 |
|------|------|
| `/help` | 显示帮助信息 |
| `/clear` | 清空当前对话 |
| `/model` | 显示当前模型信息 |
| `/quit` / `/exit` | 退出 Baize |
| `/session` | 显示当前会话 ID |
| `/sessions` | 列出历史会话 |

### 会话管理

- TUI 启动时自动创建或恢复会话（SQLite `data/baize.db`）
- 消息实时持久化：user → 执行前写入，assistant → 完成后写入
- `--resume <id>` 标志恢复指定会话
- `/sessions` 列出最近会话，`/session <id>` 切换

### 状态栏

```
Baize v0.4   sess-abc123 · deepseek-chat   steps: 3/30   tokens: 1,234   ◉ READY
```

- 左侧：版本号
- 中部：会话 ID + 模型名
- 右侧：步数 / 最大步数 + Token 累计 + 模式指示器

---

## 状态管理

### MessageState：图内状态

每个 weave 图实例持有一份 `*MessageState`，随节点执行逐步演化。

```
MessageState
├── Messages []Message      会话消息列表（LLM 历史上下文）
│   ├── RoleUser            用户输入
│   ├── RoleAssistant       LLM 回复（含 ToolCalls）
│   ├── RoleTool            工具执行结果
│   └── RoleSystem          系统提示词
├── Context map[string]any  节点间传递数据（RAG 检索结果等）
├── StepCount int           已执行的 LLM 步数
├── MaxSteps int            步数上限（防止无限循环）
├── TotalTokens int         累计 Token 消耗
├── CurrentAgent string     Supervisor 当前路由目标
├── NextAgent string        下一跳 Agent 名
└── CompletedAgents []string 已完成子 Agent 列表
```

### 会话持久化：跨次运行

```
SQLite (data/baize.db)
├── sessions
│   ├── id, title, model, workspace
│   ├── step_count, total_tokens
│   ├── status (active/paused/completed/aborted)
│   └── created_at, updated_at
└── messages
    ├── session_id  ──FK── sessions(id) ON DELETE CASCADE
    ├── role, content
    ├── tool_calls  JSON
    └── created_at
```

会话生命周期：

```
POST /chat (无 session_id) → 创建 Session + 追加 user message
POST /chat (带 session_id) → 加载 Session + 恢复历史 Messages + 追加 user message
Agent RunStream → 流式输出
EventDone         → 保存 assistant message 到 Session
GET /sessions/{id} → 完整对话历史
```

### 消息管理

| 组件 | 位置 | 策略 |
|------|------|------|
| **图内消息** | `MessageState.Messages` | 每次 LLM 调用追加 assistant，每次工具执行追加 tool。不做截断 |
| **短期记忆** | `memory/ShortTermMemory` | 环形缓冲区：超出上限时逐出最旧的非系统消息，保护 system prompt |
| **长期记忆** | `memory/LongTermMemory` | Embed → VectorStore 语义检索，独立于会话生命周期 |
| **长期记忆（文件）** | `memory/MarkdownStore` | Markdown 文件 + YAML frontmatter，关键词匹配检索 |

---

## 工具系统与扩展

### 注册与发现

```
                     ToolRegistry
                          │
          ┌───────────────┼───────────────┐
          │               │               │
    6 个内置工具       MCP Manager     未来扩展
    ┌──────────┐    ┌──────────────┐
    │ file     │    │ Server A     │
    │ shell    │    │ (filesystem) │
    │ git      │    ├──────────────┤
    │ web_search│   │ Server B     │
    │ web_fetch│    │ (database)   │
    │ calculator│   └──────────────┘
    └──────────┘
```

### Tool / SafeTool 接口

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]any     // JSON Schema
    Execute(ctx, args) (string, error)
}

type SafeTool interface {
    Tool
    IsReadOnly() bool                // 是否仅读取，不修改外部状态
    RequiredPermissions() []Permission // file:read | shell:exec | git:write ...
    AffectedPaths(args) []string     // 受影响的文件路径（用于路径级权限判定）
}
```

ToolNode 在执行前检查 `SafeTool` 接口：如果工具实现了它，则提取权限元数据交给权限引擎决策；否则按 `IsReadOnly=true` 处理（允许执行但记录审计）。

### MCP 管理

MCP Manager 负责第三方工具服务的完整生命周期：

```
MCP Manager
├── AddServer(name, command, args...)   启动子进程，MCP 握手，发现工具
├── RemoveServer(name)                  关闭子进程
├── Tools() []Tool                      聚合所有 server 的工具
├── Servers() []string                  列出已注册 server
└── Close()                             关闭所有子进程

每个 MCP Server 对应一个 ClientAdapter：
  stdio 子进程 ──JSON-RPC── ClientAdapter ──包装为── []tool.Tool
```

启动流程：`AddServer → 创建 ClientAdapter → adapter.Connect() → Initialize + ListTools → 聚合到 ToolRegistry`

---

## 多智能体调度

### Supervisor Agent

```
                          Supervisor (LLM)
                               │
          ┌────────────────────┼────────────────────┐
          │ route 工具调用      │                    │
          ▼                    ▼                    ▼
   ReActAgent("search")  ReActAgent("edit")  ReActAgent("test")
          │                    │                    │
          ▼                    ▼                    ▼
     独立图执行            独立图执行            独立图执行
          │                    │                    │
          └────────────────────┼────────────────────┘
                               │
                          collect 节点
                          (状态合并)
                               │
                          Supervisor 汇总 → 最终回答
```

调度方式：**顺序路由**。Supervisor LLM 每次调用一个 route 工具，指定目标子 Agent。该子 Agent 在自己的 weave 图中完整运行完毕后，状态合并回 Supervisor，Supervisor 决定下一步（继续路由或结束）。当前为顺序执行，后续规划并行调度。

### 任务状态追踪

- `MessageState.CurrentAgent` — 当前正在执行的子 Agent 名
- `MessageState.CompletedAgents` — 已完成子 Agent 列表
- `MessageState.StepCount` — 全局步数计数器
- `MessageState.Context` — 跨 Agent 数据传递通道

---

## 权限模型

权限检查位于 ToolNode 内部，在 `tool.Execute()` 之前执行，不可绕过：

```
ToolNode.executeToolCall()
  │
  ├── 提取 SafeTool 元数据
  │   ├── RequiredPermissions
  │   ├── AffectedPaths
  │   └── 命令模式（shell 工具）
  │
  ▼
PermissionChecker.CheckPermission(toolName, args)
  │
  ├── allow → tool.Execute() → 返回结果
  ├── deny  → 返回 "Permission denied: {reason}"
  └── ask   → 返回 "Confirmation required"（TUI 弹窗 / API 返回确认请求）
```

### 策略引擎

按优先级匹配 `PolicyRule` 列表，命中即停：

- `PathPattern`（glob）+ `Command` + `Domain` → `Decision`（Allow/Deny/Ask）
- 支持 `ScopeAlways` 学习决策：用户的一次选择可跨会话生效
- `DefaultPolicy()` 预置规则：读文件允许、`rm`/`sudo` 拒绝、git 写入询问、API 域名白名单

### 审计

`AuditLogger` 记录每次权限决策的时间戳、上下文和理由，用于追溯。

---

## 沙箱

当前工具在 Agent 进程内同步执行。规划中通过 [Carrel](https://github.com/wzhongyou/carrel) 项目提供系统级隔离：

```
ToolNode.executeToolCall()
  │
  ├── 当前：tool.Execute()  进程内同步执行
  │
  └── 规划：Carrel Sandbox
      ├── 独立 cgroup/namespace
      ├── 文件系统隔离
      ├── 网络策略
      └── 资源限制
```

`ShellTool` 已内置一层软防护：屏蔽 `rm -rf /`、`dd`、fork 炸弹等危险命令，超时 120s，环境变量精简。

---

## 异步队列（规划）

当前 Agent 执行是同步的：一次 `/chat` 请求阻塞直到 Agent Loop 完成（通过 SSE 流式输出中间结果）。规划引入异步任务队列：

```
POST /chat  →  返回 task_id（立即）
                →  入队 → Worker 执行
                →  SSE / WebSocket 推送进度
                →  GET /tasks/{id} 查询状态/结果
```

---

## 包依赖

```
baize ──▶ weave      图执行引擎（节点调度、条件分支、Hook 体系）
       ──▶ llmgate    LLM 多模型网关（20+ 提供商、路由、降级）
       ──▶ mcp-go     MCP 协议（JSON-RPC over stdio）
       ──▶ sqlite     纯 Go SQLite（会话存储，无 CGO）
       ──▶ bubbletea  TUI 框架（全屏终端交互）
```

## 包分层

```
cli/                   CLI 入口——组装组件，启动三种模式
  cli/tui/             Bubble Tea 全屏终端

core/                  AI 引擎
  core/agent/          Agent 引擎
    core/agent/llmgate/  llmgate 适配器（LLMModel 接口实现）
  core/tool/           工具接口 + ToolRegistry
    core/tool/builtin/  6 个内置工具
    core/tool/mcp/      MCP 客户端（Manager + ClientAdapter）
  core/permission/     权限引擎（策略匹配、决策学习、审计）
  core/session/        SQLite 会话持久化
  core/memory/         短期记忆（环形缓冲）+ 长期记忆（Markdown + VectorStore）
  core/context/        项目分析（语言、框架、构建工具）

server/                HTTP+SSE API 服务
  server/middleware/   请求 ID · CORS · 日志

protocol/              智能体会话协议（API 类型定义）

ide/vscode/            VSCode 插件
ide/jetbrains/         JetBrains 插件
```
