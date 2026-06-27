# Baize 会话系统技术设计

> 关联文档：[设计 V1.1](design-v1.1.md)

---

## Session Store（SQLite）

实现：`core/session/store.go`，使用 `modernc.org/sqlite`（纯 Go，零 CGO）。

### 数据模型

```sql
sessions (id, title, model, workspace, step_count, total_tokens, status, created_at, updated_at)
messages (id, session_id, role, content, tool_calls JSON, created_at)
```

### 核心 API

| 方法 | 说明 |
|------|------|
| `CreateSession(session)` | 创建新会话 |
| `GetSession(id)` | 读取会话及其所有消息 |
| `UpdateSession(session)` | 更新 title/step_count/status |
| `ListSessions()` | 按 updated_at 倒序，最多 100 条 |
| `DeleteSession(id)` | 级联删除消息 |
| `AddMessage(sessionID, msg)` | 追加消息，自动更新 updated_at |
| `GetMessages(sessionID)` | 按插入顺序返回所有消息 |

---

## 多轮历史注入

请求时从 SQLite 读历史，经过 `ContextBudget.Trim()` 后注入 `MessageState`：

```go
func buildInitialState(sessionID, userMsg string) *MessageState {
    history, _ := sessionStore.GetMessages(sessionID)
    history = contextBudget.Trim(history)         // 超预算时裁剪
    return &MessageState{
        Messages: append(history, Message{Role: "user", Content: userMsg}),
    }
}
```

每轮结束后，agent 消息和 tool_result 通过 `AddMessage` 持久化到 SQLite。

---

## Session API

### HTTP 接口（server 模式，供 VSCode 插件/Web 客户端）

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/sessions` | 列出所有会话（最近 100 条）|
| `GET` | `/api/v1/sessions/:id` | 获取会话详情 + 消息列表 |
| `DELETE` | `/api/v1/sessions/:id` | 删除会话及所有消息 |
| `POST` | `/api/v1/chat` | 创建/继续会话，流式响应（SSE）|

### CLI 直接访问（直连 session.Store）

CLI 不走 HTTP，直接操作同一个 `baize.db`：

| CLI 功能 | 调用 |
|---------|------|
| `baize --resume <id>` | `Store.GetSession(id)` 读历史，注入 MessageState |
| `baize --list` / TUI 会话列表 | `Store.ListSessions()` |
| 会话自动保存 | `Store.AddMessage()` 每步持久化 |

CLI 和 server 操作同一个数据库文件，无需协调。

---

## Checkpoint（计划中）

```
设计目标：
  - 每 N 步自动保存 checkpoint（MessageState 快照）
  - baize --resume <session_id> 恢复上次会话
  - baize --resume <session_id> --checkpoint <id> 从指定点恢复
  - 支持分支：从某个 checkpoint 开启新的独立会话

存储方案：
  - checkpoints 表（session_id, step, snapshot JSON, created_at）
  - 或直接利用 messages 表的 id 作为 checkpoint 标记
```

当前状态：`sessions` 表已有 `step_count` 字段，数据结构就绪，执行侧待实现。

> 消息队列与并发控制设计见 [server-design.md](server-design.md)。
