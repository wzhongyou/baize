# Baize Eval 评测体系设计

> 关联文档：[设计 V1.1](design-v1.1.md) | 多智能体成功率见 [multi-agent-design.md](multi-agent-design.md)

---

## 为什么需要 Eval

没有评测，就不知道每次改动是让 agent 变好还是变差。Eval 是产品质量的基础设施，优先级不低于功能开发。

Anthropic 和 OpenAI 内部对每个 prompt 变更、模型升级都有严格的回归评测。

---

## 评测维度

| 维度 | 指标 | 说明 |
|------|------|------|
| 任务成功率 | pass@1 | 一次执行通过测试用例的比例 |
| 工具调用准确率 | tool precision/recall | 调用了正确工具 + 参数正确 |
| 步数效率 | steps / task | 完成任务的平均步数（少 = 好）|
| 幻觉率 | hallucination rate | 引用了不存在的文件/函数的比例 |
| Token 效率 | tokens / task | 完成任务的平均 token 消耗 |
| 用户满意度 | thumbs up/down | 人工标注（长期）|

---

## Golden Dataset（黄金测试集）

每个任务是一个 golden example：

```yaml
# evals/golden/fix-null-pointer.yaml
id: fix-null-pointer-001
category: bug_fix
description: "修复空指针异常"
workspace: testdata/go-project/
setup:
  - copy: fixtures/buggy_main.go → main.go
input: "main.go 第 42 行有空指针异常，帮我修复"
assertions:
  - type: file_contains
    path: main.go
    pattern: "if .* != nil"
  - type: no_file_modified
    path: go.mod
  - type: exit_code
    command: "go build ./..."
    expected: 0
```

分类：
- `bug_fix`：修复明确的 bug
- `feature_add`：新增功能
- `refactor`：重构代码
- `explain`：解释代码（非破坏性）
- `multi_file`：跨多文件任务

---

## 评测运行器

```go
// evals/runner.go
type EvalRunner struct {
    agent   AgentRunner
    dataset []GoldenExample
}

func (r *EvalRunner) Run(ctx context.Context) *EvalReport {
    for _, ex := range r.dataset {
        result := r.runOne(ctx, ex)
        // 记录 pass/fail、步数、token 消耗
    }
}
```

每个 golden example：
1. 准备 workspace（复制 fixture 文件）
2. 运行 agent（`agentRunner.Run`）
3. 执行断言（文件内容、命令退出码、工具调用序列）
4. 清理 workspace

---

## Prompt 变更回归流程

每次修改 `buildSystemPrompt`、工具描述、BAIZE.md 前：

```
1. 运行全量 eval（本地）
2. 对比 pass rate 变化
3. pass rate 下降 > 2% → 不合并
4. 新功能必须附带至少 1 个 golden example
```

---

## 幻觉检测

agent 调用工具时引用了不存在的路径/符号，是常见失败模式：

```go
// PostToolUse hook 中检测
func detectHallucination(toolName string, args map[string]any, result string) bool {
    if toolName == "file_read" {
        path := args["path"].(string)
        if strings.Contains(result, "no such file") {
            return true  // LLM 捏造了文件路径
        }
    }
    return false
}
```

检测到幻觉时：记录到 eval 指标，并在下一轮 LLM 调用时追加系统提示 "该路径不存在，请先用 glob/grep 确认路径"。

---

## 人工反馈收集

TUI 中每次回答后展示简单反馈：

```
[👍 有用]  [👎 没用]  [💬 反馈]
```

反馈写入 `~/.baize/feedback.jsonl`，格式：

```json
{"session_id":"sess-xxx","message_id":"msg-yyy","rating":1,"comment":"","timestamp":"..."}
```

定期用于改进 prompt 和 golden dataset。

---

## 实现优先级

| 优先级 | 内容 |
|--------|------|
| P1 | 断言引擎（file_contains、command exit_code） |
| P1 | 首批 golden dataset（20 个覆盖主要场景） |
| P1 | CI 集成：PR 触发 eval，报告 pass rate |
| P2 | 工具调用准确率追踪 |
| P2 | 幻觉检测 hook |
| P2 | TUI 人工反馈收集 |
| P3 | Token 效率基准 |
| P3 | 自动扩充 golden dataset（从真实会话中挖取）|
