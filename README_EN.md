# Baize — The Universal Super-Agent

> "Baize knows all 11,520 creatures under heaven — what they are, and how to master them."
> An open-source **general-purpose AI agent platform**. Ask anything, automate everything, ship everywhere.

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.25-blue)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![CI](https://github.com/wzhongyou/baize/actions/workflows/ci.yml/badge.svg)](https://github.com/wzhongyou/baize/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/wzhongyou/baize)](https://goreportcard.com/report/github.com/wzhongyou/baize)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.x-blue)](https://www.typescriptlang.org/)

**Baize** is an open-source **autonomous AI agent platform** — one agent, every interface. Deep reasoning, multi-step tool execution, web research, code generation, data analysis, document creation. Access via **CLI · TUI · IDE plugin · Web Dashboard · IM Bot**.

Built from the ground up in **Go + TypeScript**, targeting parity with **Manus**, **GenSpark**, **OpenClaw**, **Codex**, and **CoWork** — Baize combines **agentic orchestration**, **multi-model LLM routing**, **OS-native sandboxing**, and **multi-channel deployment** into a single, zero-dependency binary.

```bash
brew install baize
baize "Research the latest WebAssembly edge-computing landscape and write a report"
```

---

## Why "Baize"?

In Chinese mythology, **Baize (白泽)** is a divine beast that knows all things. When the Yellow Emperor encountered Baize by the Eastern Sea, it recited the names, appearances, and weaknesses of every single one of the 11,520 supernatural creatures in the world.

That's what Baize the agent does: **you ask, it answers — and acts.**

> Keywords: `ai-agent` `autonomous-agent` `llm-agent` `multi-agent` `agent-orchestration` `tool-calling` `rag` `mcp` `model-context-protocol` `ai-automation` `developer-tools` `cli-agent` `chatbot` `coding-assistant` `general-purpose-ai` `open-source-ai` `golang` `typescript`

---

## Interface Matrix

```
CLI / TUI               IDE Plugin              Web Dashboard
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

**One agent, every surface.**

---

## Quick Start

```bash
# Install
brew install baize
# or: go install github.com/wzhongyou/baize/cmd/baize@latest

# Set your LLM API key
export ANTHROPIC_API_KEY=sk-ant-...
# or: export OPENAI_API_KEY=sk-...

# Interactive mode (TUI)
baize

# Single-shot: coding
baize "Add a health-check endpoint to this project"

# Single-shot: research (auto web-search + verification)
baize "Analyze the competitive landscape for AI coding agents in 2026"

# Single-shot: writing
baize "Write a weekly engineering status report from these bullet points"

# Resume a previous session
baize --session resume=abc123

# Launch API server + Web Dashboard
baize server --port 9779
# Open http://localhost:9779
```

**Mock mode** (no API key needed — demonstrates the full agent loop locally):
```bash
go run ./examples/agent_demo
```

---

## Architecture

```
Entry Layer (implemented)
├── CLI / TUI (Bubble Tea)      ✅ native terminal experience
├── Web Dashboard (React + TS)  ✅ visual agent management
├── API Server                  ✅ HTTP + WebSocket
├── VS Code / JetBrains         📋 planned
└── IM Gateway                  📋 planned (Telegram / Discord / Slack)

Core Engine (implemented)
├── Agent Orchestrator ✅ ReAct + Supervisor (powered by Graphflow)
├── Graphflow Engine   ✅ DAG execution, parallel nodes, streaming
├── Tool System        ✅ built-in (File/Shell/Git/Calculator) + MCP client
├── Memory             ✅ short-term + long-term interfaces
├── Session Manager    ✅ persistence + multi-session
├── Permission         ⚠️ policy engine + audit framework
└── Context Engine     ⚠️ project file analysis

Core Engine (planned)
├── Sandbox            📋 macOS Seatbelt / Linux Bubblewrap
├── Plan-Execute       📋 plan-then-execute agent mode
├── Multi-Agent        📋 agent pipelines + debate mode
├── LSP Integration    📋 Tree-sitter parsing + semantic index
└── Vector Store       📋 embeddings + hybrid search

Infrastructure (planned)
├── Scheduler          📋 Cron jobs
├── Plugin System      📋 WASM / subprocess plugins
└── Telemetry          📋 OpenTelemetry observability
```

---

## Core Capabilities

### ✅ Implemented
- **Agent Engine** — ReAct Agent Loop + Supervisor Agent on Graphflow DAG engine
- **Multi-Model** — Anthropic Claude / OpenAI GPT / Google Gemini / DeepSeek / Ollama
- **Streaming** — token-level real-time output via CLI hooks + Web SSE
- **Tool Execution** — file I/O, shell commands, git operations, calculator, MCP client
- **Session Persistence** — SQLite-backed, multi-session management
- **Web Dashboard** — React + TypeScript + Tailwind AGUI interface
- **CLI / TUI** — Bubble Tea terminal UI, interactive + single-shot modes
- **MCP Protocol** — Model Context Protocol client (tool discovery + invocation)
- **Structured Output** — JSON Schema constrained generation + validation
- **Memory System** — short-term + long-term memory interfaces
- **Permission Framework** — policy engine + audit log foundations
- **Single Binary** — Go-compiled, <100MB idle

### 📋 Planned
- **Deep Research** — web search + multi-source verification + knowledge synthesis
- **OS Sandbox** — macOS Seatbelt / Linux Bubblewrap native isolation
- **Multi-Agent** — agent collaboration, pipelines, debate
- **IM Bots** — Telegram / Discord / Slack multi-channel
- **Scheduler** — Cron jobs + async long-running agents
- **Plugin System** — WASM + subprocess plugins
- **IDE Plugins** — VS Code + JetBrains deep integration
- **Code Intelligence** — LSP + Tree-sitter indexing + hybrid search

---

## Package Layout

```
baize/
├── cmd/baize/             ✅ CLI / TUI entry point (main binary)
├── agent/                 ✅ Agent abstraction (LLM, nodes, messages, state, structured output)
├── agent/llmgate/         ✅ Multi-model LLM adapter
├── orchestrator/          ✅ Agent orchestration (ReAct + Supervisor)
├── tool/                  ✅ Tool system
├── tool/builtin/          ✅ Built-in tools (File / Shell / Git / Calculator)
├── tool/mcp/              ✅ MCP protocol (client + server)
├── context/               ⚠️ Project context (current: file analysis; planned: LSP + index)
├── session/               ✅ Session management (persistence + multi-session)
├── permission/            ⚠️ Permission system (policy engine + audit framework)
├── memory/                ⚠️ Memory system (interfaces done; planned: vector store)
├── server/                ✅ API Server (HTTP + WebSocket)
├── server/middleware/      ✅ CORS + logging middleware
├── tui/                   ✅ Terminal UI (Bubble Tea)
├── conf/                  ✅ Configuration (TOML)
├── web/                   ✅ Web Dashboard (React + TypeScript + Tailwind)
├── examples/              ✅ 5 example programs
├── docs/                  ✅ Technical documentation
│
├── sandbox/               📋 OS sandbox (planned)
├── plugin/                📋 Plugin system (planned)
├── gateway/               📋 IM gateway (planned)
├── scheduler/             📋 Job scheduler (planned)
├── ide/                   📋 IDE plugins (planned)
└── cmd/baized/            📋 Daemon mode (planned)
```

> ✅ Implemented  ⚠️ Partial  📋 Planned

---

## Documentation

- [Competitive Analysis](docs/competitive-analysis.md)
- [Architecture Design](docs/architecture.md)
- [Upgrade Roadmap](docs/upgrade-roadmap.md)
- [Subsystem Design](docs/subsystems/)

---

## Roadmap Summary

| Phase | Timeline | Focus |
|-------|----------|-------|
| **Phase 0** | ✅ Done | ReAct / RAG / Supervisor Agent + MCP + Graphflow |
| **Phase 1** | Short-term (1–2 wks) | Tools + Permissions + Sessions + CLI/TUI |
| **Phase 2** | Short-term (3–4 wks) | Sandbox + Web Dashboard + LSP Context |
| **Phase 3** | Mid-term (5–8 wks) | IM Bots + Scheduler + Plugin System |
| **Phase 4** | Long-term (9–12 wks) | IDE Deep Integration + Multi-Modal + Enterprise |

Full details: [Upgrade Roadmap](docs/upgrade-roadmap.md)

---

## Related Projects

```
Baize (白泽)                      → General-purpose super-agent (this repo)
  ├── github.com/wzhongyou/graphflow  → Graph execution engine
  └── github.com/wzhongyou/llmgate    → Multi-model LLM gateway

Cangjie (仓颉)                    → Code intelligence platform (VSCode plugin + search + agent)
```

---

## Competitors & Inspiration

Baize synthesizes the best ideas from: **Manus** (autonomous task execution), **GenSpark** (deep research synthesis), **OpenClaw** (multi-channel agent deployment), **Codex** (code-first agent loop), and **CoWork** (multi-agent collaboration patterns).

---

[MIT](LICENSE) © 2026 Wang Zhongyou
