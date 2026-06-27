# Baize Memory 系统技术设计

> 关联文档：[设计 V1.1](design-v1.1.md)

---

## 概述

Memory 系统分为三层，各自独立、按需组合：

| 层次 | 实现 | 存储 | 生命周期 |
|------|------|------|----------|
| 短期记忆 | `ShortTermMemory` | 进程内 slice | 单次会话 |
| 自动记忆 | `AutoMemory` + `memory_save` 工具 | Markdown 文件 | 跨会话持久 |
| 长期记忆 | `LongTermMemory` + VectorStore | 向量数据库 | 跨会话持久 |

---

## 短期记忆（ShortTermMemory）

进程内环形缓冲，保留最近 N 条消息（默认 100）。超出时按 FIFO 淘汰非 system 消息，system 消息永不淘汰。

```
core/memory/short_term.go
```

用途：多轮会话的滑动窗口，与 `context_budget.Trim` 协同工作。

---

## 自动记忆（AutoMemory）

Agent 在对话中发现值得记录的内容时，通过 `memory_save` 工具主动写入。

```
core/memory/auto.go
core/tool/builtin/   → memory_save 工具
```

### 目录结构

```
~/.baize/projects/<repo-slug>/memory/
  MEMORY.md         索引文件（前 200 行自动注入每次会话）
  user-prefs.md     按主题命名的记忆文件
  project-ctx.md
  <name>.md
```

### 记忆文件格式

```markdown
---
name: user-prefs
description: 用户偏好：简洁回复，不要 emoji
metadata:
  type: user
---

用户偏好简洁回复，不要在消息末尾加 emoji 或总结段落。
```

### 生效机制

1. 会话启动时自动读取 `MEMORY.md`（索引，≤200 行）注入系统提示
2. Agent 判断需要保存时调用 `memory_save(name, description, type, content)`
3. 工具写入对应 `.md` 文件并更新 `MEMORY.md` 索引

### memory_save 工具参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `name` | string | slug，如 `user-prefs` |
| `description` | string | 一行摘要，写入 MEMORY.md 索引 |
| `type` | string | `user` / `project` / `feedback` / `reference` |
| `content` | string | 记忆正文（Markdown） |

---

## 长期记忆（LongTermMemory）

基于 embedding + 向量检索的语义记忆，适合大规模知识库场景。

```
core/memory/long_term.go
```

接口依赖：

```go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float64, error)
}
type VectorStore interface {
    Insert(ctx context.Context, id string, vector []float64, metadata map[string]any) error
    Search(ctx context.Context, vector []float64, topK int) ([]SearchResult, error)
}
```

当前状态：接口已定义，具体实现（本地 SQLite-vec、Qdrant 等）作为可插拔后端，V1.1 阶段为 P3。

---

## API 层记忆接口

`POST /api/v1/memory/search` 和 `POST /api/v1/memory/save` 通过 `MarkdownStore` 实现，供外部客户端直接操作长期记忆。

```
core/memory/markdown.go   关键词匹配搜索实现
protocol/types.go         MemorySearchRequest / MemorySaveRequest
```

---

## 实现优先级

| 优先级 | 内容 |
|--------|------|
| P1 | AutoMemory + memory_save 工具 + MEMORY.md 索引注入 ✅ |
| P1 | ShortTermMemory 滑动窗口 ✅ |
| P2 | MarkdownStore 关键词搜索 API ✅ |
| P3 | LongTermMemory 向量检索后端（SQLite-vec / Qdrant） |
