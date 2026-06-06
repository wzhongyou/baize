# Baize（白泽）— 通用超级智能体

> 白泽达知万物之精，黄帝问以何术治之，一一对答。
> Baize knows all things — you ask, it answers, and acts.
> **通用超级智能体 —— 问答、编程、创作、办公，无处不在。**

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.25-blue)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Baize（白泽）** 是一个通用超级智能体平台。既能深度问答，也能自主执行复杂任务——写代码、做设计、查资料、写文档、处理数据、创作内容。通过 CLI / TUI / IDE 插件 / Web Dashboard / IM Bot 任一入口，访问同一个超级 Agent。

对标并汲取 **Manus**、**GenSpark**、**OpenClaw**、**Codex**、**CoWork** 等前沿产品的设计理念，目标成为最完整的开源通用超级智能体。

> 全 Go + TypeScript 实现。单二进制分发，零运行时依赖。
> `brew install baize` 即可开始。

---

## 为什么叫 Baize（白泽）？

中国神话中，白泽是通晓万物的神兽。黄帝在东海遇到白泽，白泽告诉他：天下有 11,520 种妖魔鬼怪，每一种叫什么、长什么样、怎么对付——全知道。

Baize 继承同一使命：**理解你的意图，掌握各类工具，自主执行复杂任务**——无论什么领域，一问便知，一言即行。

---

## 入口矩阵

```
CLI / TUI               IDE 插件                Web Dashboard
    │                       │                       │
    ▼                       ▼                       ▼
  baize "..."        VS Code / JetBrains     http://localhost:9779
    │                       │                       │
    └───────────────────────┼───────────────────────┘
                            │
                    ┌───────▼───────┐
                    │   Baize API   │
                    │(HTTP+WS+gRPC) │
                    └───────┬───────┘
                            │
              ┌─────────────┼─────────────┐
              │             │             │
        Telegram Bot   Discord Bot   Slack Bot  ...
```

---

## 安装

```bash
# macOS / Linux
brew install baize

# 或通过 go install
go install github.com/wzhongyou/baize/cmd/baize@latest
```

---

## 快速开始

```bash
# 设置 API key
export ANTHROPIC_API_KEY=sk-ant-...

# 交互式 TUI
baize

# 编程
baize "给这个项目加一个 HTTP 健康检查接口"

# 写作
baize "帮我写一份本周工作周报"

# 研究
baize "调研 WebAssembly 在边缘计算的应用现状，整理成报告"

# 办公
baize "分析这份 Excel 销售额数据，找出趋势并生成图表"

# 恢复之前的会话
baize --session resume=abc123

# 启动 API Server + Web Dashboard
baize server --port 9779
```

Mock 模式（无需 API key，演示 Agent Loop 流程）：
```bash
go run ./examples/agent_demo
```

---

## 架构

```
入口层
├── CLI / TUI（Bubble Tea）  — 终端原生体验
├── VS Code / JetBrains       — IDE 深度集成
├── Web Dashboard             — 可视化管理（React + TypeScript）
└── IM Gateway                — Telegram / Discord / Slack / WhatsApp / 微信

核心引擎
├── Agent Orchestrator — ReAct / Plan-Execute / Multi-Agent / Human-in-the-Loop
├── Graphflow 图引擎    — 节点编排、并行执行、检查点、流式输出
├── Tool System         — 内置工具 + MCP 协议 + 插件扩展
├── Context Engine      — LSP 代码理解 + 语义索引 + Git 感知
├── Sandbox             — macOS Seatbelt / Linux Bubblewrap 原生沙箱
├── Memory              — 短期/长期/事件记忆 + 向量存储
├── Permission          — 分级权限（allow/deny/ask）+ 审计日志
└── Session Manager     — 持久化、检查点、分支、上下文压缩

基础设施
├── Scheduler            — Cron 定时任务 + 异步长时 Agent
├── Plugin System        — Go / WASM / 子进程 多形态插件
└── Telemetry            — OpenTelemetry 可观测性
```

---

## 核心能力

### 通用智能
- **深度问答** — 联网搜索 + 多源验证 + 知识推理，不只是聊天
- **多模态交互** — 图片理解、文档分析、语音交互（规划中）
- **写作与创作** — 文章、报告、翻译、润色、头脑风暴
- **办公效率** — 文档处理、表格分析、邮件撰写、日程管理
- **编程开发** — 全栈代码生成、重构、调试、测试、部署

### Agent 平台
- **全 Go 技术栈** — 单二进制分发，空闲内存 < 100MB
- **多入口统一** — CLI、TUI、IDE 插件、Web Dashboard、IM Bot
- **多模型支持** — Anthropic Claude / OpenAI GPT / Gemini / DeepSeek / 本地模型
- **Agent 图编排** — 基于 Graphflow，支持 ReAct / Plan-Execute / Multi-Agent / HITL
- **流式响应** — 逐 Token 实时输出，思考过程可视化
- **会话管理** — 持久化 + 检查点 + 会话分支 + 上下文智能压缩

### 工具与安全
- **丰富工具集** — 文件 / Shell / Git / 浏览器 / 搜索 / 代码理解
- **MCP 协议** — 模型上下文协议，双向支持（客户端 + 服务端）
- **原生沙箱** — macOS Seatbelt / Linux Bubblewrap，OS 级进程隔离
- **权限系统** — allow/deny/ask 三级 + 策略引擎 + 完整审计日志
- **LSP 集成** — Tree-sitter 多语言解析 + 语义嵌入索引

### 平台与生态
- **多渠道消息** — Telegram / Discord / Slack / WhatsApp / 微信 Bot
- **定时调度** — Cron 定时任务 + 异步长时 Agent 执行
- **插件生态** — WASM 插件 + 子进程插件，社区可扩展
- **审计透明** — 每步操作有 diff 可审查，完整操作日志

---

## 包结构

```
baize/
├── cmd/baize/             # CLI/TUI 入口（主二进制）
├── cmd/baized/            # 守护进程 / API Server
├── agent/                 # Agent 抽象层（LLM、节点、消息、状态、结构化输出）
├── orchestrator/          # Agent 编排器（ReAct / Plan / Multi / HITL）
├── tool/                  # 工具系统（内置工具 + MCP 协议 + 插件注册）
├── sandbox/               # OS 级沙箱（Seatbelt + Bubblewrap）
├── context/               # 项目上下文引擎（LSP + 索引 + Git 感知）
├── session/               # 会话管理（持久化 + 检查点 + 分支）
├── permission/            # 权限系统（策略引擎 + 审计日志）
├── memory/                # 记忆系统（短期 + 长期 + 向量存储）
├── server/                # API Server（HTTP + WebSocket + gRPC）
├── tui/                   # 终端 UI（Bubble Tea 框架）
├── plugin/                # 插件系统（WASM + 子进程）
├── gateway/               # 多渠道消息网关
├── scheduler/             # 调度系统（Cron + 异步执行）
├── conf/                  # 配置系统（TOML）
├── web/                   # Web Dashboard（React + TypeScript + Vite）
├── ide/                   # IDE 插件（VS Code + JetBrains）
├── examples/              # 示例程序
└── docs/                  # 技术文档
```

---

## 文档

- [竞品分析](docs/competitive-analysis.md)
- [架构设计](docs/architecture.md)
- [升级路线图](docs/upgrade-roadmap.md)
- [子系统设计](docs/subsystems/)

---

## 路线图（概要）

| 阶段 | 周期 | 目标 |
|------|------|------|
| **Phase 0** | ✅ 完成 | ReAct / RAG / Supervisor Agent + MCP + Graphflow 集成 |
| **Phase 1** | 短期（1-2 周） | 工具系统 + 权限 + 会话持久化 + CLI/TUI 可用 |
| **Phase 2** | 短期（3-4 周） | OS 沙箱 + Web Dashboard + LSP 代码理解 |
| **Phase 3** | 中期（5-8 周） | 多渠道 IM Bot + 调度系统 + 插件体系 |
| **Phase 4** | 长期（9-12 周） | IDE 深度集成 + 多模态 + 企业级特性 |

详见 [详细路线图](docs/upgrade-roadmap.md)。

---

## 相关项目

```
Baize（白泽）                  → 通用超级智能体（本仓库）
  ├── github.com/wzhongyou/graphflow  → 图执行引擎
  └── github.com/wzhongyou/llmgate    → LLM 多模型网关

Cangjie（仓颉）                 → 代码智能平台（VSCode 插件 + 代码搜索 + Agent）
```

---

[MIT](LICENSE) © 2026 Wang Zhongyou
