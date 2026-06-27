# Baize Context 管理技术设计

> 关联文档：[设计 V1.1](design-v1.1.md)

---

## Context 预算管理

### 预算分配

| 区域 | 占比 | token 数（128K 模型） |
|------|------|-----------------------|
| 系统提示词 | 5% | ~6 400 |
| 工具定义 | 10% | ~12 800 |
| 历史消息 | 60% | ~76 800 |
| 当前轮次输入 | 15% | ~19 200 |
| 输出预留 | 10% | ~12 800 |

历史消息超出 60% 预算时触发截断或压缩。

### DefaultContextBudget().Trim() 策略

实现位于 `core/agent/context_budget.go`：

```
1. system 消息永远保留
2. 非 system 消息从最新开始向前累积，直到预算耗尽
3. 若最近消息仍超预算，截断最老保留消息的 content 以适配
```

默认 `MaxHistoryTokens = 60_000`，token 估算：`len(content) / 4`。

---

## 滚动压缩（计划中）

> 参考：Claude Code `/compact`、OpenCode `overflow.ts`、Codex `CompactionStrategy::Memento`

### 竞品策略对比

| | Claude Code | OpenCode | Codex CLI |
|-|-------------|----------|-----------|
| 触发阈值 | 80% context | `model.limit - 20K` | 90% context |
| 保留尾部 | 最近 2 轮（2K~8K tokens） | 最近若干轮 | 最近用户消息（≤20K tokens） |
| 工具结果 | 不传摘要 LLM，从摘要+尾部重建 | 截断到 2000 字符 | **完全丢弃** |
| 摘要格式 | 增量更新前次摘要 | 结构化节（Goal/Progress/Key Decisions） | Handoff 风格（进展+待办） |
| 图片 | 压缩前剥离 | `stripMedia` 剥离 | 完全丢弃 |
| 特殊保护 | Pre/PostCompact hooks | skill 工具结果永不裁剪 | 系统提示重新注入 |

### Baize 压缩策略

**触发**：`estimateTokens(history) > contextBudget * 0.75`，或阶段边界（更可预测）。

**执行步骤**：

```
1. 保留最近 10 轮原文（tail，编程任务局部性强）
2. 对 tail 之前的消息：
   - tool_call：保留（token 少，但记录了"做了什么"）
   - tool_result > 500 字符：截断到 500 字符（skill 工具结果除外，不裁剪）
   - assistant 推理：传给压缩 LLM 生成摘要
   - 图片：剥离
3. 压缩 LLM 生成结构化摘要（见下方格式）
4. 摘要以 role=system 插入历史起始处，替换原始消息
5. BAIZE.md 从磁盘重新注入（不依赖 context 内的副本）
```

**摘要格式**（参考 OpenCode 结构化节）：

```
[Context Summary]
Progress: 已完成 explore 阶段，分析了 main.go、handler.go；确认变更计划（3个文件）
Key Decisions: 使用 JWT 而非 session token；不改动 db 层
Remaining Tasks: 修改 middleware.go、handler.go；跑测试
Critical Context: auth 模块在 core/auth/，gin 框架，gorm v2
Failed Paths: 尝试过在 handler 层做鉴权，因循环依赖放弃
```

**失败保护**：连续 3 次压缩失败后停止自动压缩，降级为 Trim（简单丢弃旧消息）。

---

## KV Cache 优先设计

消息序列固定前缀，最大化 KV Cache 命中率：

```
[0] system: 核心指令（最稳定，每次不变）
[1] system: 工具定义（工具增删才变化）
[2] system: BAIZE.md 项目规则
[3..N] 历史消息（只追加，绝不修改）
[N+1] user: 当前输入
```

规则：
- 已有消息绝不修改，只追加
- 工具定义 JSON key 排序确定（map → sorted slice）
- 摘要块插入位置固定（历史起始处）

---

## todo.md 注意力机制

Agent 在多步骤任务中自动维护 `.baize/todo.md`：

```markdown
# 当前任务
- [x] 读取 main.go 理解入口
- [ ] 修复多轮历史注入 bug
- [ ] 补充单元测试
```

每次 LLM 调用前，todo.md 内容注入 context 末尾（追加位置，不破坏 KV Cache 前缀）。

---

## BAIZE.md 指令系统

### 四级加载

```
优先级（低 → 高，全部拼接）：
  ~/.baize/BAIZE.md          用户全局规则（编码风格、语言偏好）
  <project>/BAIZE.md         项目规则（构建命令、架构约定）
  <project>/BAIZE.local.md   本地覆盖（不提交 git）
  .baize/rules/*.md          路径级规则（懒加载，计划中）
```

实现：`core/context/instructions.go` `InstructionLoader.Load()`

### 路径级懒加载（计划中）

```yaml
# .baize/rules/go-style.md frontmatter
---
paths:
  - "**/*.go"
  - "go.mod"
---
```

只有 Agent 读取匹配路径的文件时，该规则才注入 context。大型 monorepo 下节省 token。

### 加载时机

会话启动时由 `buildSystemPrompt` 读取，注入 `[2] system: BAIZE.md` 位置。压缩后从磁盘重读。
