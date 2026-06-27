# Baize 多智能体系统设计

> 关联文档：[设计 V1.1](design-v1.1.md) | 实现：`core/agent/agents.go`

---

## 参考协议

| 项目 | 参考点 |
|------|--------|
| Claude Code | explore 子 agent 上下文隔离、只读轻模型分离 |
| Manus | 阶段化执行、文件系统作为外部记忆、滚动摘要 |
| OpenCode | SubAgent 接口抽象、Supervisor 路由模式 |

---

## 智能体类型

| 类型 | 工具权限 | 模型策略 | 用途 |
|------|----------|----------|------|
| `ReActAgent` | 全部 | 主模型 | 单轮编程任务，当前主力 |
| `explore` | 只读（file_read, grep, glob） | 轻量模型 | 代码搜索理解，独立 context，不污染主 agent |
| `plan` | 只读 | 主模型（思考模式） | 任务分解、变更计划生成 |
| `edit` | file_*, shell | 主模型 | 代码变更执行 |
| `test` | shell, file_read | 主模型 | 测试运行、验证 |
| `SupervisorAgent` | 仅 route 工具 | 主模型 | 多子 agent 路由调度 |

`explore`/`plan`/`edit`/`test` 均是 `ReActAgent` 的配置变体，共用同一实现，通过工具权限和模型配置区分角色。

---

## 架构：Supervisor 模式

```
SupervisorAgent（主）
  │  route(agent, task)
  ├──→ explore   独立 context，只读，返回摘要
  ├──→ plan      只读，返回结构化变更计划
  ├──→ edit      执行变更，返回 diff 摘要
  └──→ test      验证，返回 pass/fail + 详情
```

**上下文隔离原则（当前实现的缺陷修复）**：

子 agent 在独立 MessageState 中运行，完成后只把**最终 assistant 消息**作为 `tool_result` 返回给主 agent。主 agent context 不包含子 agent 的中间推理过程。

```go
// 修复前（错误）：*s = *subResult.FinalState  // 污染主 context
// 修复后（正确）：
lastMsg := lastAssistant(subResult.FinalState)
s.Messages = append(s.Messages, Message{
    Role: RoleTool, Content: lastMsg.Content,
    ToolCallID: tc.ID, ToolName: agentName,
})
```

---

## 长任务：阶段化执行

长任务（>10 步）的核心问题：context 线性增长、单点失败代价高、LLM 注意力漂移。

### 阶段划分

```
Phase 1  explore   理解代码结构，输出关键文件列表和问题定位
Phase 2  plan      生成结构化变更计划，用户确认后推进
Phase 3  edit      按计划执行变更（每文件一个子任务）
Phase 4  test      运行测试验证，输出 pass/fail
```

每个阶段结束打 Checkpoint，失败从上一阶段末恢复，不从头重跑。

### 阶段间压缩

每个阶段结束是天然的压缩点（比动态阈值检测更可预测）。触发 `context-design.md` 中定义的滚动压缩，把该阶段的中间推理替换为结构化摘要：

```
[Phase 1 摘要] 分析完成：入口 main.go:42，发现问题：auth 无 JWT 验证。
[Phase 2 摘要] 变更计划已确认：新增 jwt.go，修改 middleware.go、handler.go。
[Phase 3 当前] 正在执行 edit...
```

context 大小与阶段数无关，稳定在固定预算内。压缩策略详见 [context-design.md](context-design.md)。

### todo.md 阶段状态

```markdown
## 长任务：添加 JWT 鉴权

### Phase 1: Explore ✅
- [x] 分析入口和 handler 结构
- [x] 定位现有 auth 实现

### Phase 2: Plan ✅
- [x] 生成变更计划（用户已确认）

### Phase 3: Edit 🔄
- [x] 新增 core/auth/jwt.go
- [ ] 修改 server/middleware.go
- [ ] 修改 server/handler.go

### Phase 4: Test ⏳
```

---

## 成功率保障：验证门

每个阶段结束强制验证，不通过则重试，超限则暂停等待用户介入。

```
phase N 完成
    │
    ▼
[verification gate]
    ├── pass → phase N+1
    ├── fail (retry_count < 3) → 重试 phase N
    └── fail (retry_count ≥ 3) → 暂停，SSE 推送 user_intervention 事件
```

验证逻辑按阶段类型：
- `edit` 后：检查文件变更是否与计划一致，是否通过语法检查
- `test` 后：检查测试命令退出码，解析失败用例

**成功率估算**：

| 方案 | 单步成功率 | 30步总成功率 |
|------|-----------|-------------|
| 无保障 | 95% | 21% |
| 阶段 Checkpoint（4阶段） | 95% | ~46% |
| + 验证门（单步提升到99%） | 99% | 74% |
| + 滚动压缩（防 context 失忆） | 99.5% | 86% |

验证门是单一收益最高的机制。

---

## 任务状态机

```
Task
  ├── status: pending | running | succeeded | failed | retrying | paused
  ├── phase:  explore | plan | edit | test
  ├── subtasks: []SubTask
  │     ├── agent_type: string
  │     ├── status: pending | running | succeeded | failed
  │     ├── result: string
  │     ├── error: string
  │     └── retry_count: int
  └── checkpoint_id: string
```

### 状态流转

```
pending
  → running（开始执行）
      → succeeded（所有阶段通过验证）
      → failed（超过重试上限）
      → paused（需要用户介入）
          → running（用户确认后继续）
      → retrying（验证失败，自动重试）
          → running（重试开始）
```

### SSE 进度事件

```json
{"type": "phase_start", "phase": "edit",  "phase_index": 3, "total_phases": 4}
{"type": "phase_done",  "phase": "plan",  "summary": "变更计划：3个文件"}
{"type": "verification_fail", "phase": "edit", "retry": 1, "reason": "test syntax error"}
{"type": "user_intervention", "phase": "edit", "reason": "超过最大重试次数，需要人工介入"}
{"type": "phase_progress", "phase": "edit", "done": 1, "total": 3}
```

客户端据此渲染多阶段进度条和干预提示。

---

## 并行子任务

Supervisor 目前顺序调度，扩展支持批量 `route`：

```go
// route 工具支持 agents 数组
// Supervisor 发起并行分发
{"agents": ["explore_auth", "explore_db"], "task": "分析两个模块"}
```

`supervisorRouteNode` 起多个 goroutine，等所有子 agent 完成后把各自结果作为多条 `tool_result` 返回。适用场景：多文件并行分析、多模块独立修改（无依赖时）。

---

## 预算感知

Agent 启动时估算任务预算，接近上限主动触发压缩：

```go
const (
    BudgetWarningRatio  = 0.75  // 达到 75% 触发压缩
    BudgetCriticalRatio = 0.90  // 达到 90% 降级（跳过非关键步骤）
)
```

每阶段开始前检查剩余预算，不足以完成当前阶段时暂停并通知用户，不允许 context 无限增长。

---

## 实现优先级

| 优先级 | 内容 | 状态 |
|--------|------|------|
| P0 | 修复 Supervisor 上下文污染（子 agent 只返回最终结果） | 待修复 |
| P1 | explore 内置子 agent（只读 + 轻模型配置） | 待实现 |
| P1 | 验证门（`collect` 节点加结果校验 + 重试） | 待实现 |
| P1 | 阶段化执行 + 阶段间滚动压缩 | 待实现 |
| P1 | SSE 阶段进度事件 | 待实现 |
| P2 | user_intervention 暂停机制 | 待实现 |
| P2 | 并行子任务（route 支持 agents 数组） | 待实现 |
| P2 | 预算感知压缩 | 待实现 |
| P3 | 任务状态持久化到 SQLite | 待实现 |
