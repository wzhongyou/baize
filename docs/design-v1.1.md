# Baize V1.1 系统设计

> 基于 V1.0 现状，参考 Claude Code、OpenCode、Codex CLI、Manus 最佳实践，面向"中文最好的开源编程智能体"目标的完整重设计。

---

## 设计目标

1. **能力对齐**：核心编码任务（理解、搜索、编辑、测试）达到 Claude Code 水准
2. **中文生态**：深度适配国内 LLM（DeepSeek、通义、Kimi、文心）及工作流
3. **开发者生产力**：个人和团队的第一编程工具，每天都想用
4. **开源可扩展**：社区可以贡献工具、规则、提示词，不被厂商绑定

---

## V1.0 关键缺陷（必须修复）

| 缺陷 | 影响 |
|------|------|
| 多轮历史未注入 MessageState | Agent 每轮失忆，无法完成跨轮任务 |
| file edit 是朴素字符串替换 | 多处匹配时误改，大文件改错无感知 |
| 没有内容搜索工具 | 不知道路径就找不到代码 |
| context 无截断无压缩 | 长会话必然超出 token 限制 |
| 并行工具执行是死代码 | 多工具任务比应有速度慢 |
| 长期记忆只有接口无实现 | 记忆系统完全不工作 |
| 会话只写不读 | 历史会话无法恢复继续 |

---

## 核心设计原则（V1.1）

| 原则 | 说明 |
|------|------|
| **文件系统是外部记忆** | 超出 context 的内容压缩存文件，随时可恢复（来自 Manus） |
| **KV Cache 优先** | context 只追加，工具定义放前，序列化确定性（来自 Manus） |
| **错误证据保留** | 失败的工具调用留在 context，不清除，帮助模型适应（来自 Manus） |
| **子 Agent 上下文隔离** | 探索类任务用独立轻量子 Agent，不污染主 context（来自 CC） |
| **按需加载规则** | 指令文件按路径匹配懒加载，不全量注入（来自 CC） |
| **单二进制分发** | musl 静态链接，goreleaser 跨平台构建，curl 一行安装 |

---

## 架构总览（V1.1）

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              客户端层                                    │
│  ┌──────────┐  ┌──────────┐  ┌────────────┐  ┌───────────────────────┐ │
│  │   TUI    │  │  VSCode  │  │ JetBrains  │  │  CLI / Pipe / CI      │ │
│  │ BubbleTea│  │Extension │  │  Plugin    │  │  baize "query" | grep │ │
│  └────┬─────┘  └────┬─────┘  └─────┬──────┘  └──────────┬────────────┘ │
└───────┼─────────────┼──────────────┼─────────────────────┼──────────────┘
        └─────────────┴──────────────┴─────────────────────┘
                                     │
┌────────────────────────────────────┼────────────────────────────────────┐
│                          API 层 (server/)                               │
│  HTTP + SSE  localhost:9779                                             │
│  /health  /chat  /sessions/*  /memory/*  /tools/*                      │
│  Middleware: RequestID · CORS · Logging · RateLimit                    │
└────────────────────────────────────┼────────────────────────────────────┘
                                     │
                              AgentRunner
                                     │
┌────────────────────────────────────┼────────────────────────────────────┐
│                        Agent 引擎 (core/agent/)                         │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │                    主 Agent Loop (ReAct)                          │  │
│  │                                                                  │  │
│  │  [规划节点] → [LLM节点] ──HasToolCalls?──→ [工具节点] → 循环    │  │
│  │      ↑           ↑                              │                │  │
│  │      │      KV Cache 优先的消息构建              │                │  │
│  │      │      todo.md 注意力注入                   │                │  │
│  │      └──────────────────────────────────────────┘                │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────────────┐   │
│  │  Explore Agent  │  │  Plan Agent     │  │  Supervisor Agent    │   │
│  │  轻模型·只读    │  │  只读·思考      │  │  多子Agent路由调度   │   │
│  │  独立context    │  │  独立context    │  │  顺序+并行混合       │   │
│  └─────────────────┘  └─────────────────┘  └──────────────────────┘   │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
┌────────────────────────────────────┼────────────────────────────────────┐
│                     Context 管理层 (core/context/)                      │
│                                                                         │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────────┐ │
│  │  指令文件加载器   │  │  Context 预算管理 │  │  项目分析            │ │
│  │  BAIZE.md 多层级 │  │  token 计数+截断  │  │  读 go.mod/pkg.json  │ │
│  │  路径级懒加载    │  │  滚动压缩摘要    │  │  依赖感知·框架识别   │ │
│  └──────────────────┘  └──────────────────┘  └──────────────────────┘ │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
┌────────────────────────────────────┼────────────────────────────────────┐
│                        工具系统 (core/tool/)                            │
│                                                                         │
│  ┌───────────────────────────── ToolRegistry ────────────────────────┐ │
│  │  内置工具 (9个)  +  MCP Manager  +  Hooks (PreTool/PostTool)      │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                                                                         │
│  file_read     file_write    file_edit(diff)   file_list               │
│  grep          glob          shell             git                     │
│  web_search    web_fetch     calculator                                │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
┌────────────────────────────────────┼────────────────────────────────────┐
│                     状态与记忆层                                         │
│                                                                         │
│  MessageState    Session(SQLite)   BAIZE.md       MarkdownMemory       │
│  多轮历史正确注入  会话可读可写      项目指令文件    agent自动写记忆       │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 模块详细设计

### 1. 多轮会话 — 修复 Agent 失忆

**问题**：V1.0 每次请求重建 Agent，历史消息从未注入 `MessageState`。

**V1.1 设计**：

```go
// AgentRunner 初始化时加载历史
func (r *agentRunner) RunStream(ctx, req, onEvent) {
    msgs := r.sessionStore.GetMessages(req.SessionID)  // 从 SQLite 读历史
    state := &MessageState{
        Messages: append(msgs, Message{Role: "user", Content: req.Message}),
    }
    // 构建 agent，注入完整历史
}
```

历史注入前先经过 **Context 预算管理**（见第3节）裁剪，避免超出 token 限制。

---

### 2. 工具系统 — 9 个内置工具

#### 2.1 新增 grep 工具

```
grep(pattern, path, [flags])
  - 使用 ripgrep 或 grep -r 递归搜索
  - 返回：文件路径:行号:匹配内容
  - 支持正则、大小写忽略、文件类型过滤
  - 输出截断：最多 200 行，超出提示 "use --include or narrow pattern"
```

#### 2.2 file_edit 升级为 diff 模型

```
file_edit(path, old_string, new_string)
  规则：
  1. old_string 必须在文件中唯一出现，否则报错并列出所有匹配行号
  2. 支持多 hunk：传入 edits: [{old, new}, ...] 数组，原子应用
  3. 编辑前自动备份到 .baize/edit_backups/
  4. 返回 unified diff 供 LLM 确认
```

#### 2.3 file_read 支持行范围

```
file_read(path, [start_line], [end_line])
  - 无范围：读全文，超过 500 行时返回前 500 行 + "file truncated" 提示
  - 有范围：精确读取，用于大文件定向读取
```

#### 2.4 工具命名约定（来自 Manus）

工具按功能域加前缀，支持组级权限控制：

| 前缀 | 工具 | 权限域 |
|------|------|--------|
| `file_` | file_read, file_write, file_edit, file_list | file |
| `shell_` | shell_exec | shell |
| `git_` | git_status, git_diff, git_commit... | git |
| `web_` | web_search, web_fetch | web |
| (无) | grep, glob, calculator | readonly |

#### 2.5 工具输出截断

所有工具在结果注入 MessageState 前进行截断：

```go
const MaxToolOutputTokens = 4000

func truncateToolOutput(output string) string {
    if estimateTokens(output) <= MaxToolOutputTokens {
        return output
    }
    // 保留头尾，中间插入截断提示
    return head + "\n...[output truncated, " + size + " total]...\n" + tail
}
```

#### 2.6 Hooks 系统（来自 Claude Code）

```go
type HookEvent string
const (
    PreToolUse   HookEvent = "pre_tool_use"
    PostToolUse  HookEvent = "post_tool_use"
    SessionStart HookEvent = "session_start"
    SessionStop  HookEvent = "session_stop"
)

type Hook struct {
    Event   HookEvent
    Matcher string   // 工具名 glob，"file_*" 匹配所有文件工具
    Command string   // shell 命令，exit 2 = 硬拒绝，exit 0 = 放行
}
```

配置在 `.baize/hooks.toml`，支持：
- 编辑后自动 gofmt/prettier
- 提交前自动 lint
- 审计日志写到外部系统

---

### 3. Context 管理 — 不再 OOM

#### 3.1 Token 预算管理

```
总预算: model.ContextWindow (e.g. 128K)
分配：
  系统提示词       5%  (固定，含 BAIZE.md 核心规则)
  工具定义         10% (固定前缀，保证 KV Cache 命中)
  历史消息         60% (可压缩)
  当前轮次输入      15%
  输出预留         10%

当历史消息超出 60% 预算时，触发滚动压缩。
```

#### 3.2 滚动压缩（来自 Claude Code /compact）

```
触发条件：历史消息 token > budget * 0.6
压缩策略：
  1. 保留最近 N 轮完整消息（N = 10）
  2. 对更早的消息调用 LLM 生成摘要
  3. 摘要替换原始消息，插入 role=system 摘要块
  4. 文件内容等大块输出：保留路径引用，丢弃内容（可重新读取）

BAIZE.md 内容在压缩后重新注入（从磁盘重读，不依赖 context 内的副本）。
```

#### 3.3 KV Cache 优先设计（来自 Manus）

```
消息序列固定前缀顺序（保证 KV Cache 命中率最大化）：
  [0] system: 核心指令（最稳定）
  [1] system: 工具定义（次稳定，工具增删才变化）
  [2] system: BAIZE.md 项目规则
  [3..N] 历史消息（追加不修改）
  [N+1] user: 当前输入

规则：
  - 已有消息绝不修改，只追加
  - 工具定义 JSON key 排序确定性（map → sorted slice）
  - 摘要块插入位置固定（历史起始处）
```

#### 3.4 todo.md 注意力机制（来自 Manus）

Agent 在执行多步骤任务时，自动维护 `.baize/todo.md`：

```markdown
# 当前任务
- [x] 读取 main.go 理解入口
- [x] 找到 handler_chat.go
- [ ] 修复多轮历史注入 bug
- [ ] 补充单元测试
```

每次 LLM 调用前，将 todo.md 内容注入 context 末尾（非前缀，避免破坏 KV Cache）。

---

### 4. 指令文件系统 — BAIZE.md（来自 CC 的 CLAUDE.md）

#### 4.1 四级加载

```
优先级从低到高：
  ~/.baize/BAIZE.md          用户全局规则（编码风格、语言偏好）
  <project>/BAIZE.md         项目规则（构建命令、架构约定）
  <project>/BAIZE.local.md   本地覆盖（不提交 git，个人偏好）
  .baize/rules/*.md          路径级规则（见下）
```

#### 4.2 路径级懒加载

```yaml
# .baize/rules/go-style.md
---
paths:
  - "**/*.go"
  - "go.mod"
---
Go 代码必须通过 golangci-lint。
错误处理使用 fmt.Errorf("context: %w", err) 包装。
```

只有当 Agent 读取匹配路径的文件时，该规则才注入 context。大型 monorepo 下节省大量 token。

#### 4.3 Agent 自动写记忆

Agent 可以写入 `~/.baize/projects/<repo>/memory/` 目录：

```
memory/
  MEMORY.md          索引文件（前 200 行自动加载）
  user_prefs.md      用户偏好
  project_context.md 项目背景
  <topic>.md         按主题分类
```

每次会话启动时自动加载 MEMORY.md + 相关 topic 文件。

---

### 5. 子 Agent 架构（来自 CC + OpenCode）

#### 5.1 内置子 Agent

| Agent | 模型策略 | 工具权限 | 用途 |
|-------|---------|---------|------|
| `explore` | 优先轻量模型 | 只读 (file_read, grep, glob) | 代码搜索、理解，不污染主 context |
| `plan` | 主模型 | 只读 | 任务分解、方案设计 |
| `edit` | 主模型 | 全部 | 代码编写和修改 |
| `test` | 主模型 | shell_exec + file | 跑测试、验证 |

#### 5.2 调用方式

- 主 Agent 通过 `delegate(agent, task)` 工具调用子 Agent
- 子 Agent 在独立的 `MessageState` 中运行，完成后返回结果字符串
- 主 Agent context 只看到子 Agent 的最终结果，不包含中间推理过程

#### 5.3 模型路由

```toml
# BAIZE.md 或 llmgate.toml 配置
[agents.explore]
model = "deepseek-chat"      # 便宜快速

[agents.edit]
model = "deepseek-r1"        # 主力模型

[agents.plan]
model = "qwen-max-thinking"  # 思考模型
```

---

### 6. 权限系统升级

#### 6.1 三档模式（来自 Codex CLI）

```
suggest    只读模式：展示计划，不执行任何写操作
auto-edit  自动编辑文件，shell 命令逐条确认
full-auto  全自动，权限规则控制边界
```

启动参数：`baize --mode suggest`

#### 6.2 细粒度 Glob 权限（来自 OpenCode）

```toml
# .baize/settings.toml
[permissions]
allow = [
  "file_read:**",
  "file_edit:src/**",
  "shell_exec:go test ./...",
  "shell_exec:go build ./...",
  "git_*:*",
]
deny = [
  "shell_exec:rm *",
  "shell_exec:curl *",
]
ask = [
  "file_edit:*.toml",
  "file_edit:*.yaml",
]
```

---

### 7. 项目分析升级

V1.0 只看顶层文件名。V1.1 读文件内容做深度分析：

```go
// 读 go.mod 获取真实模块路径和依赖
// 读 package.json 获取 scripts、dependencies
// 读 Makefile 提取构建/测试命令
// 读 .github/workflows/*.yml 了解 CI 流程
```

分析结果注入系统提示词，并建议写入 BAIZE.md 供后续会话复用。

---

### 8. 会话系统修复

#### 8.1 历史正确注入

```go
func buildInitialState(sessionID string, userMsg string) *MessageState {
    history := sessionStore.GetMessages(sessionID)
    history = contextBudget.Trim(history)  // 超预算时压缩
    return &MessageState{
        Messages: append(history, userMsg),
    }
}
```

#### 8.2 Checkpoint（激活已有数据结构）

```go
// 每 N 步自动保存 checkpoint
// 支持 baize --resume <session_id> --checkpoint <id>
// 支持分支：从某个 checkpoint 开启新会话
```

---

### 9. 分发与发布

#### 9.1 goreleaser 跨平台构建

```yaml
# .goreleaser.yml
builds:
  - goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
    env:
      - CGO_ENABLED=0   # 纯 Go SQLite，零依赖
```

#### 9.2 安装方式

```bash
# curl 一行安装
curl -fsSL https://baize.run/install.sh | sh

# Homebrew
brew install wzhongyou/tap/baize

# Go install
go install github.com/wzhongyou/baize/cli@latest
```

#### 9.3 版本管理

- 版本号单一来源：`internal/version/version.go`
- ldflags 在构建时注入 `Version`、`Commit`、`BuildDate`
- git tag `v*` 触发 CI 自动发布

---

## 实现优先级

### P0 — 让 Agent 真正可用

1. 修复多轮历史注入（session → MessageState）
2. 增加 grep 工具
3. file_edit 升级为 diff 模型 + 唯一性校验
4. 工具输出截断
5. 会话历史可恢复（读 SQLite 并注入）

### P1 — 达到生产质量

6. Context 预算管理 + 滚动压缩
7. BAIZE.md 多级加载（用户/项目/local）
8. todo.md 注意力机制
9. Hooks 系统（pre/post tool）
10. 并行工具执行（修复死代码）
11. 子 Agent 架构（explore 轻模型）

### P2 — 差异化与生态

12. 路径级规则懒加载
13. goreleaser + 自动发布
14. 三档权限模式
15. 项目分析深度读配置文件
16. Agent 自动写记忆
17. Checkpoint 激活

---

## 与 V1.0 的接口兼容性

- API 接口（`/chat`、`/sessions/*`）保持兼容，新增字段向后兼容
- `AgentRunner` 接口不变，内部实现替换
- `BAIZE.md` 是新概念，不影响现有用户
- 工具名增加前缀（`file_read` 替代 `file`），LLM 自动适配，提示词同步更新
