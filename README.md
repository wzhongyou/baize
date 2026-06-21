# Baize（白泽）— 通用超级智能体

> 白泽达知万物之精，黄帝问以何术治之，一一对答。
> Baize knows all things — you ask, it answers, and acts.
> **通用超级智能体 —— 问答、编程、创作、办公，无处不在。**

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.25-blue)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/wzhongyou/baize)](https://goreportcard.com/report/github.com/wzhongyou/baize)
[![CI](https://github.com/wzhongyou/baize/actions/workflows/ci.yml/badge.svg)](https://github.com/wzhongyou/baize/actions/workflows/ci.yml)

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
入口层（已实现）
├── CLI / TUI（Bubble Tea）  ✅ 终端原生体验
├── Web Dashboard             ✅ 可视化管理（React + TypeScript）
├── API Server                ✅ HTTP + WebSocket
├── VS Code / JetBrains       📋 规划中
└── IM Gateway                📋 规划中（Telegram / Discord / Slack）

核心引擎（已实现）
├── Agent Orchestrator ✅ ReAct Agent + Supervisor（基于 Graphflow）
├── Graphflow 图引擎    ✅ 节点编排、并行执行、流式输出
├── Tool System         ✅ 内置工具（File/Shell/Git/Calculator）+ MCP 客户端
├── Memory              ✅ 短期记忆 + 长期记忆接口
├── Session Manager     ✅ 持久化 + 多会话
├── Permission          ⚠️ 基础权限（策略引擎 + 审计日志框架）
└── Context Engine      ⚠️ 项目文件分析

核心引擎（规划中）
├── Sandbox             📋 macOS Seatbelt / Linux Bubblewrap
├── Plan-Execute Agent  📋 规划-执行模式
├── Multi-Agent 协作     📋 多 Agent 管道 + 辩论模式
├── LSP 代码智能         📋 Tree-sitter 解析 + 语义索引
└── Vector Store        📋 向量嵌入 + 混合检索

基础设施（规划中）
├── Scheduler            📋 Cron 定时任务
├── Plugin System        📋 WASM / 子进程插件
└── Telemetry            📋 OpenTelemetry 可观测性
```

---

## 核心能力

### ✅ 已实现
- **Agent 引擎** — ReAct Agent Loop + Supervisor Agent，基于 Graphflow 图执行
- **多模型支持** — Anthropic Claude / OpenAI GPT / Gemini / DeepSeek / Ollama
- **流式响应** — 逐 Token 实时输出，CLI Hook + Web SSE
- **工具执行** — 文件读写、Shell 命令、Git 操作、计算器、MCP 客户端
- **会话持久化** — SQLite 存储，多会话管理
- **Web Dashboard** — React + TypeScript + Tailwind，AGUI 界面
- **CLI / TUI** — Bubble Tea 终端 UI，交互式 + 单次模式
- **MCP 协议** — 模型上下文协议客户端（工具发现 + 调用）
- **结构化输出** — JSON Schema 约束 + 校验
- **记忆系统** — 短期记忆接口 + 长期记忆接口
- **权限框架** — 策略引擎 + 审计日志基础
- **单二进制** — Go 编译，空闲 < 100MB

### 📋 规划中
- **深度研究** — 联网搜索 + 多源验证 + 知识推理
- **OS 沙箱** — macOS Seatbelt / Linux Bubblewrap 原生隔离
- **Multi-Agent** — 多 Agent 协作、流水线、辩论
- **IM Bot** — Telegram / Discord / Slack 多渠道
- **定时调度** — Cron 任务 + 异步长时 Agent
- **插件系统** — WASM + 子进程插件
- **IDE 插件** — VS Code + JetBrains 深度集成
- **代码智能** — LSP 集成 + Tree-sitter 索引 + 混合搜索

---

## 包结构

```
baize/
├── cmd/baize/             ✅ CLI/TUI 入口（主二进制）
├── agent/                 ✅ Agent 抽象层（LLM、节点、消息、状态、结构化输出）
├── agent/llmgate/         ✅ LLM 多模型适配器
├── orchestrator/          ✅ Agent 编排器（ReAct + Supervisor）
├── tool/                  ✅ 工具系统
├── tool/builtin/          ✅ 内置工具（File / Shell / Git / Calculator）
├── tool/mcp/              ✅ MCP 协议（客户端 + 服务端）
├── context/               ⚠️ 项目上下文（当前：文件分析；规划：LSP + 索引）
├── session/               ✅ 会话管理（持久化 + 多会话）
├── permission/            ⚠️ 权限系统（策略引擎 + 审计框架）
├── memory/                ⚠️ 记忆系统（接口完成；规划：向量存储）
├── server/                ✅ API Server（HTTP + WebSocket）
├── server/middleware/      ✅ CORS + 日志中间件
├── tui/                   ✅ 终端 UI（Bubble Tea）
├── conf/                  ✅ 配置（TOML）
├── web/                   ✅ Web Dashboard（React + TypeScript + Tailwind）
├── examples/              ✅ 5 个示例程序
├── docs/                  ✅ 技术文档
│
├── sandbox/               📋 OS 沙箱（规划中）
├── plugin/                📋 插件系统（规划中）
├── gateway/               📋 IM 网关（规划中）
├── scheduler/             📋 调度系统（规划中）
├── ide/                   📋 IDE 插件（规划中）
└── cmd/baized/            📋 守护进程模式（规划中）
```

> ✅ 已实现  ⚠️ 部分实现  📋 规划中

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
  ├── github.com/wzhongyou/weave      → 图执行引擎
  └── github.com/wzhongyou/llmgate    → LLM 多模型网关

Cangjie（仓颉）                 → 代码智能平台（VSCode 插件 + 代码搜索 + Agent）
```

---

[MIT](LICENSE) © 2026 Wang Zhongyou
