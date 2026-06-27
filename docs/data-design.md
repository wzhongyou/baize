# Baize 数据存储设计

> 关联文档：[设计 V1.1](design-v1.1.md)

---

## 存储介质总览

| 介质 | 路径 | 存储内容 |
|------|------|----------|
| SQLite | `{data_dir}/baize.db` | 会话、消息、Checkpoint |
| 文件系统（Markdown） | `~/.baize/projects/{repo}/memory/` | Agent 自动记忆 |
| 文件系统（Markdown） | `~/.baize/skills/{name}/SKILL.md` | 已安装 Skill 定义 |
| 文件系统（Markdown） | `{workspace}/.baize/todo.md` | 当前任务进度 |
| 文件系统（Markdown） | `BAIZE.md` / `~/.baize/BAIZE.md` | 项目/用户指令规则 |

SQLite 使用 WAL 模式 + 外键约束，纯 Go 驱动（`modernc.org/sqlite`），零 CGO 依赖。

---

## 实体与表结构

### 1. Session（会话）

一次与 agent 的完整对话上下文。

```sql
CREATE TABLE sessions (
    id           TEXT PRIMARY KEY,              -- "sess-{nano timestamp}"
    title        TEXT NOT NULL DEFAULT '',      -- 取首条消息前 50 字
    model        TEXT NOT NULL DEFAULT '',      -- 使用的模型，如 "deepseek-r1"
    workspace    TEXT NOT NULL DEFAULT '',      -- 绝对路径，如 "/home/user/proj"
    step_count   INTEGER NOT NULL DEFAULT 0,   -- agent 执行步数
    total_tokens INTEGER NOT NULL DEFAULT 0,   -- 累计 token 消耗
    status       TEXT NOT NULL DEFAULT 'active', -- active | paused | completed | aborted
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT NOT NULL DEFAULT (datetime('now'))
);
```

**status 状态流转**：`active` → `completed`（正常结束）/ `aborted`（用户中断）/ `paused`（计划中）

---

### 2. Message（消息）

会话内的每一条消息，按插入顺序构成多轮历史。

```sql
CREATE TABLE messages (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id   TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role         TEXT NOT NULL,        -- user | assistant | system | tool
    content      TEXT NOT NULL DEFAULT '',
    tool_calls   TEXT DEFAULT NULL,    -- JSON: [{id, name, arguments}]，仅 assistant 消息
    images       TEXT DEFAULT NULL,    -- JSON: ["data:image/png;base64,..."]，仅 user 消息
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_messages_session ON messages(session_id);
```

**role 说明**：
- `user`：用户输入，可携带 images
- `assistant`：LLM 回复，可携带 tool_calls
- `tool`：工具执行结果，对应某个 tool_call 的 id
- `system`：系统提示、BAIZE.md 内容（不持久化，运行时注入）

**tool_calls JSON 结构**：
```json
[{"id": "call-abc", "name": "file", "arguments": {"action": "read", "path": "main.go"}}]
```

---

### 3. Checkpoint（快照，计划中）

会话执行过程中的状态快照，用于回放和分支。

```sql
CREATE TABLE checkpoints (
    id           TEXT PRIMARY KEY,           -- "ckpt-{nano timestamp}"
    session_id   TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    step_count   INTEGER NOT NULL,           -- 对应 session.step_count
    message_id   INTEGER REFERENCES messages(id), -- 快照到哪条消息为止
    state        TEXT NOT NULL,              -- JSON: 序列化的 MessageState
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_checkpoints_session ON checkpoints(session_id);
```

**当前状态**：数据结构已在 `session.go` 定义，store 层待实现。

---

### 4. Branch（分支，计划中）

从某个 Checkpoint 开启的新会话分支。

```sql
CREATE TABLE branches (
    id            TEXT PRIMARY KEY,
    parent_id     TEXT NOT NULL REFERENCES sessions(id),
    checkpoint_id TEXT NOT NULL REFERENCES checkpoints(id),
    session_id    TEXT NOT NULL REFERENCES sessions(id), -- 分支对应的新 session
    name          TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);
```

---

## 文件系统存储

### Agent 自动记忆

```
~/.baize/projects/{repo-slug}/memory/
  MEMORY.md          索引文件（前 200 行注入每次会话）
  {name}.md          各条记忆，YAML frontmatter + Markdown 正文
```

**记忆文件格式**：
```markdown
---
name: user-prefs
description: 用户偏好：简洁回复，不加 emoji
metadata:
  type: user
---

回复保持简洁，不在末尾加总结段落。
```

详见 [memory-design.md](memory-design.md)。

---

### Skill 定义

```
~/.baize/skills/{name}/
  SKILL.md           frontmatter（name/description/triggers）+ 系统提示正文
  mcp.json           可选，MCP server 定义
```

详见 [skill-design.md](skill-design.md)。

---

### 任务进度

```
{workspace}/.baize/todo.md
```

Agent 在多步任务中维护，格式为标准 Markdown 任务列表：

```markdown
- [x] 读取 main.go 理解入口
- [ ] 修复多轮历史注入 bug
- [ ] 补充单元测试
```

每次 LLM 调用前注入 context 末尾，执行后由 agent 更新。不持久化到 SQLite，生命周期与工作区绑定。

---

### 项目/用户指令规则

```
~/.baize/BAIZE.md              用户全局规则
{workspace}/BAIZE.md           项目规则
{workspace}/BAIZE.local.md     本地覆盖（不提交 git）
{workspace}/.baize/rules/*.md  路径级规则（按文件匹配懒加载）
```

只读，不由 agent 修改（除非用户显式要求）。

---

## 数据流：一次完整请求

```
POST /chat
  │
  ├─ 读 sessions（有无 session_id）
  ├─ 读 messages（历史，经 ContextBudget.Trim 裁剪）
  ├─ 写 messages（user 消息，立即持久化）
  │
  ├─ agent 执行中...
  │     每步写 messages（tool_call、tool_result、assistant）
  │     更新 sessions.step_count / total_tokens
  │
  └─ done
        写 messages（最终 assistant 回复）
        更新 sessions.status / updated_at
```

---

## 实现优先级

| 优先级 | 内容 |
|--------|------|
| P0 | sessions + messages 表，基础 CRUD ✅ |
| P0 | 文件系统：memory、todo.md ✅ |
| P1 | messages 表新增 images 字段 |
| P2 | checkpoints 表 + Checkpoint API |
| P2 | branches 表 + Branch API |
