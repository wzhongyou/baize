# Baize V1.1 系统设计索引

> 基于 V1.0 现状，面向"中文最好的开源编程智能体"目标的完整重设计。

---

## 设计目标

1. **能力对齐**：核心编码任务（理解、搜索、编辑、测试）达到 Claude Code 水准
2. **中文生态**：深度适配国内 LLM（DeepSeek、通义、Kimi、文心）及工作流
3. **开发者生产力**：个人和团队的第一编程工具，每天都想用
4. **开源可扩展**：社区可以贡献工具、规则、提示词，不被厂商绑定

---

## V1.0 关键缺陷（必须修复）

| 缺陷                        | 影响                             |
| --------------------------- | -------------------------------- |
| 多轮历史未注入 MessageState | Agent 每轮失忆，无法完成跨轮任务 |
| file edit 是朴素字符串替换  | 多处匹配时误改，大文件改错无感知 |
| 没有内容搜索工具            | 不知道路径就找不到代码           |
| context 无截断无压缩        | 长会话必然超出 token 限制        |
| 并行工具执行是死代码        | 多工具任务比应有速度慢           |
| 长期记忆只有接口无实现      | 记忆系统完全不工作               |
| 会话只写不读                | 历史会话无法恢复继续             |

---

## 核心设计原则

| 原则                                              | 来源        |
| ------------------------------------------------- | ----------- |
| 文件系统是外部记忆，超出 context 的内容压缩存文件 | Manus       |
| KV Cache 优先：工具定义放前，消息只追加           | Manus       |
| 错误证据保留：失败的工具调用留在 context          | Manus       |
| 子 Agent 上下文隔离：探索类任务独立轻量 Agent     | Claude Code |
| 按需加载规则：BAIZE.md 路径匹配懒加载             | Claude Code |
| 单二进制分发：musl 静态链接，goreleaser 跨平台    | -           |

---

## 架构总览

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
└────────────────────────────────────┼────────────────────────────────────┘
                                     │
                              AgentRunner
                                     │
┌────────────────────────────────────┼────────────────────────────────────┐
│                        Agent 引擎 (core/agent/)                         │
│  [规划节点] → [LLM节点] ──HasToolCalls?──→ [工具节点] → 循环           │
│  KV Cache 优先消息构建  ·  todo.md 注意力注入                           │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
┌────────────────────────────────────┼────────────────────────────────────┐
│                     Context 管理层 (core/context/)                      │
│  BAIZE.md 多层级加载  ·  Context 预算管理  ·  项目分析                  │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
┌────────────────────────────────────┼────────────────────────────────────┐
│                        工具系统 (core/tool/)                            │
│  内置工具 (9个)  +  MCP Manager  +  Hooks (PreTool/PostTool)           │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
┌────────────────────────────────────┼────────────────────────────────────┐
│                     状态与记忆层                                         │
│  MessageState  ·  Session(SQLite)  ·  BAIZE.md  ·  MarkdownMemory      │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 子系统设计文档

| 文档                                            | 内容                                                                   |
| ----------------------------------------------- | ---------------------------------------------------------------------- |
| [tui-cli-design.md](tui-cli-design.md)             | TUI（Bubble Tea）与 CLI 设计：flags、斜线命令、流式渲染、权限确认 UI   |
| [vscode-extension-design.md](vscode-extension-design.md) | VSCode 插件：WebView、SSE 客户端、IDE 上下文注入、权限弹窗        |
| [jetbrains-plugin-design.md](jetbrains-plugin-design.md) | JetBrains 插件：Kotlin、JCef WebView、OkHttp SSE、Action 注册    |
| [agent-engine-design.md](agent-engine-design.md)   | Agent 引擎框架：ReActAgent、节点类型、Hook、执行流程、用户确认         |
| [sandbox-design.md](sandbox-design.md)             | 沙箱设计：carrel 集成、macOS Seatbelt、Linux bubblewrap+seccomp        |
| [llm-gateway-design.md](llm-gateway-design.md)     | 模型网关：llmgate 现状、竞品对比、成本追踪、Prompt Cache、负载均衡     |
| [multi-agent-design.md](multi-agent-design.md)     | 多智能体架构、长任务阶段化执行、验证门、任务状态机                     |
| [chat-protocol-design.md](chat-protocol-design.md) | 会话协议：ChatRequest/ChatEvent 字段说明、SSE 事件流、富内容块、错误码 |
| [data-design.md](data-design.md)                   | 数据存储：实体设计、SQLite 表结构、文件系统布局、数据流                |
| [tool-design.md](tool-design.md)                   | 三层工具架构、ToolRegistry、9 个内置工具、权限系统、Hooks              |
| [context-design.md](context-design.md)             | Context 预算、Trim 策略、KV Cache 设计、todo.md、BAIZE.md              |
| [session-design.md](session-design.md)             | SQLite 会话存储、多轮历史注入、Session API、Checkpoint                 |
| [server-design.md](server-design.md)               | HTTP + SSE 运行时、请求调度、Session 级并发控制                        |
| [memory-design.md](memory-design.md)               | 短期/自动/长期记忆三层架构、memory_save 工具                           |
| [mcp-design.md](mcp-design.md)                     | MCP 双向集成：Client（消费者）+ Server（提供者）                       |
| [skill-design.md](skill-design.md)                 | Skill 能力包、两级加载机制、activate_skill 工具                        |

---

## 实现优先级

### P0 — 让 Agent 真正可用

1. 修复多轮历史注入（session → MessageState）
2. 增加 grep 工具
3. file_edit 唯一性校验
4. 工具输出截断
5. 会话历史可恢复

### P1 — 达到生产质量

6. Context 预算管理 + 滚动压缩
7. BAIZE.md 多级加载
8. todo.md 注意力机制
9. Hooks 系统
10. 并行工具执行（修复死代码）
11. 子 Agent 架构（explore 轻模型）

### P2 — 差异化与生态

12. 路径级规则懒加载
13. goreleaser + 自动发布
14. 三档权限模式
15. 项目分析深度读配置文件
16. Agent 自动写记忆
17. Checkpoint 激活
18. Skill 系统（详见 [skill-design.md](skill-design.md)）

---

## 与 V1.0 的接口兼容性

- API 接口（`/chat`、`/sessions/*`）保持兼容，新增字段向后兼容
- `AgentRunner` 接口不变，内部实现替换
- 工具名增加前缀（`file_read` 替代 `file`），LLM 自动适配
- `BAIZE.md` 是新概念，不影响现有用户
