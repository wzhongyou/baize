# Baize TUI & CLI 技术设计

> 关联文档：[设计 V1.1](design-v1.1.md) | Agent 引擎见 [agent-engine-design.md](agent-engine-design.md)

---

## CLI 设计

### 入口与启动模式

`cli/main.go`，三种运行模式：

| 命令 | 模式 | 说明 |
|------|------|------|
| `baize` | TUI 交互 | 默认，启动 Bubble Tea |
| `baize --no-tui` | 简单 REPL | 无 TUI，stdin/stdout 交互 |
| `baize "question"` | 单次执行 | 非交互，输出后退出 |
| `baize server` | API Server | 启动 HTTP + SSE 服务 |

### 核心 Flags

| Flag | 默认 | 说明 |
|------|------|------|
| `-config` | `conf/llmgate.toml` | LLM 配置文件路径 |
| `-provider` | auto | 覆盖 provider |
| `-model` | 默认 | 覆盖模型 ID |
| `-workspace` | `.` | 工作区根目录 |
| `-max-steps` | 30 | agent 最大执行步数 |
| `-mode` | `auto-edit` | 权限模式：`suggest`/`auto-edit`/`full-auto` |
| `-verbose` | false | 详细日志到 stderr |
| `-no-tui` | false | 禁用 TUI，使用简单 REPL |
| `-port`/`-host` | 9779/127.0.0.1 | server 模式监听地址 |

### 斜线命令

TUI 内 `/` 前缀触发，有自动补全提示：

| 命令 | 说明 |
|------|------|
| `/help` | 显示帮助 |
| `/quit` `/exit` | 退出 |
| `/clear` | 清屏 |
| `/model <id>` | 切换模型 |
| `/workspace <path>` | 切换工作区 |

### 数据流

```
用户输入 → textarea → Enter → handleSubmit
                                   │
                                   ▼
                           startAgent() goroutine
                                   │
                           tuiAgentRunner.RunStream
                                   │
                           chan tea.Msg (buffer=64)
                                   │
                           waitForEvent → Bubble Tea Update
                                   │
                           handleStreamEvent → m.messages
                                   │
                           chatView() re-render
```

---

## TUI 设计

### 技术栈

| 库 | 版本 | 用途 |
|----|------|------|
| `github.com/charmbracelet/bubbletea` | v1.3.10 | TUI 框架（Elm 架构） |
| `github.com/charmbracelet/bubbles` | v1.0.0 | textarea、viewport 组件 |
| `github.com/charmbracelet/lipgloss` | v1.1.0 | 样式渲染 |

### Bubble Tea Model 结构

```go
type Model struct {
    mode        uiMode          // startup | input | streaming | confirm
    messages    []ChatMsg       // 对话历史（渲染用）
    viewport    viewport.Model  // 滚动消息区
    textarea    textarea.Model  // 输入框
    streaming   bool
    thinkingBuf strings.Builder // 思考内容累积
    permConfirm chan bool        // 权限确认通道
    permTool    string          // 当前待确认工具名
    history     []string        // 输入历史（up/down 导航）
    totalSteps  int
    totalTokens int
}
```

### UI 模式状态机

```
startup ──(选择开始)──→ input
                          │
                    (提交消息)
                          │
                          ▼
                      streaming ──(done)──→ input
                          │
                    (permission_ask)
                          │
                          ▼
                       confirm ──(y/n/a)──→ streaming
```

### 流式事件渲染

| 事件 type | 渲染行为 |
|-----------|---------|
| `thought` | 追加到 `thinkingBuf`，显示为思考指示器 |
| `tool_call` | 追加工具调用气泡，显示工具名+参数 |
| `tool_result` | 填充上一条工具调用的结果 |
| `answer` | 追加 assistant 消息，清空 `thinkingBuf` |
| `done` | 显示 token/步数统计，恢复输入 |
| `error` | 显示错误消息 |

### 权限确认 UI

```
⚠ Agent 请求执行：shell_exec
  命令：rm -rf dist/

[Y] 允许  [N] 拒绝  [A] 始终允许  [Esc] 取消
```

- `Y` → 发 `true` 到 `permConfirm` channel
- `N`/`Esc` → 发 `false`
- `A` → 发 `true` + 调用 `policyEngine.Learn(ScopeAlways)` 持久化

### 会话管理

当前 TUI 每次启动为新会话，历史保存在 `m.messages` 内存中，通过 `tuiAgentRunner.RunStream` 的 `history` 参数传递给 agent。

计划中（P1）：startup 界面显示历史会话列表，支持选择恢复，调用 `session.Store.ListSessions()` 直连数据库。

---

## 实现优先级

| 优先级 | 内容 | 状态 |
|--------|------|------|
| P0 | TUI 基础交互 + 流式渲染 + 权限确认 | ✅ |
| P0 | 斜线命令 + 自动补全 | ✅ |
| P1 | startup 界面会话列表 + 恢复 | 待实现 |
| P1 | `/skills` 命令列出已安装 skill | 待实现 |
| P2 | 多会话 tab | 待实现 |
| P2 | 图片粘贴上传（base64 → images 字段） | 待实现 |
