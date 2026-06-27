# Baize 工具系统技术设计

> 关联文档：[设计 V1.1](design-v1.1.md)

---

## 三层架构

```
内置工具 (builtin)
    ↓
MCP 工具 (core/tool/mcp/)      → ToolRegistry → agent LLM tool_call
    ↓
Skill 工具 (via skill.Manager)
```

所有工具统一实现 `tool.Tool` 接口，注册到 `ToolRegistry`。

---

## 核心接口

```go
// core/tool/tool.go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]any         // JSON Schema
    Execute(ctx context.Context, args map[string]any) (string, error)
}

type SafeTool interface {
    Tool
    IsReadOnly() bool
    RequiredPermissions() []Permission
    AffectedPaths(args map[string]any) []string
}
```

`ToolRegistry` 提供 `Register(t Tool)` / `Get(name)` / `List()` 方法。

---

## 9 个内置工具

| 工具 | 文件 | 说明 |
|------|------|------|
| `file` | builtin/file.go | 读/写/编辑/列目录/glob 搜索，action 参数区分 |
| `grep` | builtin/grep.go | ripgrep/grep -r 递归正则搜索，返回 `path:行号:内容` |
| `shell` | builtin/shell.go | 执行 shell 命令，超时可配 |
| `git` | builtin/git.go | git 常用操作（status/diff/commit 等）|
| `web_search` | builtin/web.go | 调用搜索 API，返回摘要列表 |
| `web_fetch` | builtin/web.go | 获取 URL 内容，HTML 转纯文本 |
| `calculator` | builtin/calculator.go | 数学表达式求值 |
| `memory_save` | builtin/memory.go | 写 Markdown 记忆文件（见 memory-design.md）|
| `activate_skill` | core/skill/ | 按需加载 Skill 完整提示词（见 skill-design.md）|

### file_edit 唯一性校验

```go
count := strings.Count(content, oldStr)
if count > 1 {
    // 报错，列出所有匹配行号，要求 LLM 提供更多上下文
    return "", fmt.Errorf("old_string matches %d times (lines: %s) — provide more context", count, lineNums)
}
```

### 工具输出截断

所有工具结果在注入 MessageState 前统一截断（`core/agent/nodes.go`）：

```go
const maxToolOutputChars = 20000  // ~5K tokens

func truncateToolOutput(s string) string {
    // 保留头部 + 尾部，中间插入截断提示
    return head + "\n...[output truncated: N chars total]...\n" + tail
}
```

---

## 权限系统

### 三档模式

| 模式 | 行为 | PermissionChecker |
|------|------|-------------------|
| `suggest` | 只读，拒绝所有写/执行 | `ReadOnlyChecker()` |
| `auto-edit` | 文件自动编辑，shell 需确认 | `PolicyEngine.AsAgentChecker()` |
| `full-auto` | 全自动，仅 deny 规则生效 | `PolicyEngine.AsAgentCheckerFullAuto()` |

### PolicyEngine + Glob 规则

```toml
# .baize/settings.toml
[permissions]
allow = ["file_read:**", "file_edit:src/**", "shell_exec:go test ./..."]
deny  = ["shell_exec:rm *", "shell_exec:curl *"]
ask   = ["file_edit:*.toml", "file_edit:*.yaml"]
```

决策优先级：deny > ask > allow。`SafeTool.AffectedPaths()` 提供路径信息供 glob 匹配。

---

## Hooks 系统

配置文件：`.baize/hooks.toml`

```toml
[[hooks]]
event   = "post_tool_use"
matcher = "file_*"        # 工具名 glob
command = "gofmt -w ."   # exit 2 = 硬拒绝，exit 0 = 放行

[[hooks]]
event   = "pre_tool_use"
matcher = "shell"
command = "echo $TOOL_ARGS | audit-logger"
```

| 事件 | 时机 |
|------|------|
| `pre_tool_use` | 工具执行前，exit 2 可阻止执行 |
| `post_tool_use` | 工具执行后，可触发格式化/lint |
| `session_start` | 会话开始 |
| `session_stop` | 会话结束 |

---

## 工具执行流程

```
LLM 返回 tool_calls
    ↓
PermissionChecker.CheckPermission(toolName, args)
    → deny: 返回错误给 LLM
    → ask:  暂停等待用户确认
    → allow: 继续
    ↓
PreToolUse Hooks
    ↓
Tool.Execute(ctx, args)
    ↓
truncateToolOutput(result)
    ↓
PostToolUse Hooks
    ↓
注入 MessageState 作为 tool_result
```

并行工具执行：多个 tool_call 在同一轮时并发执行（`errgroup`），结果按原序注入。
