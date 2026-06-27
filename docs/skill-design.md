# Baize Skill 系统技术设计

> 关联文档：[设计 V1.1](design-v1.1.md)

---

## 概述

Skill 是可安装的能力包，允许用户和社区为 Baize agent 扩展垂直领域能力。一个 Skill 包含：

- **系统提示片段**：注入 agent 上下文的领域指令
- **MCP 服务器**（可选）：提供专用工具的外部进程
- **元数据**：名称、描述、触发词，用于按需激活

Skill 与 MCP 的区别：MCP 只提供工具，Skill 同时提供提示词 + 工具，是更高层的能力抽象。

---

## 目录结构

```
~/.baize/skills/
  <skill-name>/
    SKILL.md       必须。frontmatter + 系统提示正文
    mcp.json       可选。该 skill 附带的 MCP server 定义列表
```

### SKILL.md 格式

```markdown
---
name: aigc
description: AIGC 图片生成能力，支持文生图、图生图
slash: false
triggers: 生成图片, 画图, 文生图
---

你可以使用 generate_image 工具生成图片。
调用时需要提供 prompt（英文描述）和可选的 style 参数。
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | Skill 标识符，默认为目录名 |
| `description` | string | 一行摘要，永远在 context 中（索引） |
| `slash` | bool | 是否暴露为 `/skill-name` 斜线命令 |
| `triggers` | []string | 关键词列表，匹配时提示 LLM 激活 |

frontmatter 下方的 Markdown 正文是完整的系统提示片段，**按需加载**，不默认注入。

### mcp.json 格式

```json
[
  {
    "name": "server",
    "command": "npx",
    "args": ["-y", "@aigc/mcp-server"]
  }
]
```

---

## 两级加载机制

上下文污染是 Skill 系统的核心约束。解决方案：将 Skill 内容分为两级。

### 第一级：索引（always in context）

每次 agent 启动时，将所有已安装 Skill 的 `name + description` 压缩为一行注入系统提示：

```
已安装 Skills（使用 activate_skill 工具按需加载完整指令）：aigc（AIGC 图片生成）、code-review（代码审查）
```

消耗 context 固定，不随 Skill 数量线性增长。

### 第二级：激活（on demand）

LLM 判断用户意图匹配某 Skill 后，调用内置工具 `activate_skill(name)`，tool result 返回该 Skill 的完整 `SKILL.md` 正文。LLM 收到后就地理解并使用——完整内容只在需要时进入 context。

触发路径（任一满足）：
1. **LLM 自决策**：语义匹配用户意图后主动调用 `activate_skill`
2. **用户斜线命令**：`/aigc` 直接触发（需 `slash: true`）
3. **关键词匹配**：用户输入命中 `triggers` 词组（客户端预筛，透传给 LLM）

---

## 核心实现

```
core/skill/
  skill.go      Skill 结构体、Load()（解析单个目录）
  manager.go    Manager：Load、Start、Close、SystemPromptIndex、Tools
```

### Manager 生命周期

```go
sm := skill.NewManager(skillsDir)
sm.Load(skillsDir)     // 扫描目录，解析所有 SKILL.md
sm.Start(ctx)          // 启动 MCP servers（每个 Skill 只启动一次）
defer sm.Close()       // 进程退出时关闭所有 MCP 进程
```

`Manager` 是进程级单例，在 `main` / `runServer` 入口创建，通过参数注入 `buildToolRegistry` 和 `buildSystemPrompt`，避免重复启动 MCP server。

### activate_skill 工具

```go
// 实现 tool.Tool 接口，注册到 ToolRegistry
type activateSkillTool struct{ m *Manager }

func (t *activateSkillTool) Execute(_ context.Context, args map[string]any) (string, error) {
    name := args["name"].(string)
    s := t.m.find(name)
    return s.Prompt, nil  // 返回 SKILL.md 正文，进入 tool_result
}
```

返回值直接作为 `tool_result` 消息注入 MessageState，无需改动 agent 核心。

---

## 与 MCP 的关系

```
Skill
 ├── 提示词片段（系统提示注入）
 └── mcp.json → MCP Server → 工具（注册到 ToolRegistry）
```

Skill 的 MCP 工具与内置工具、独立 MCP 工具完全同等，权限系统、工具截断、并行执行均透明适用。

---

## 富内容渲染

Skill 的 MCP 工具可返回 `__baize_blocks` 信封触发客户端富内容渲染：

```json
{"__baize_blocks": [{"type": "image", "data": "base64...", "meta": {"fallback_text": "生成的图片"}}]}
```

客户端按 `type` 渲染；未知 type 降级到 `meta.fallback_text`。详见[会话协议](design-v1.1.md)。

---

## 实现优先级

| 优先级 | 内容 |
|--------|------|
| P1 | SKILL.md 解析、Manager 生命周期、activate_skill 工具、两级加载 ✅ |
| P2 | `/skill-name` 斜线命令支持 |
| P2 | `baize skill install <url>` 远端安装 |
| P3 | `triggers` 关键词客户端预筛 |
| P3 | Skill 市场（index.json 托管） |
