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
Entry Layer
├── CLI / TUI (Bubble Tea)      — native terminal experience
├── VS Code / JetBrains         — deep IDE integration
├── Web Dashboard (React + TS)  — visual agent management
└── IM Gateway                  — Telegram / Discord / Slack / WhatsApp / WeChat

Core Engine
├── Agent Orchestrator  — ReAct / Plan-Execute / Multi-Agent / Human-in-the-Loop
├── Graphflow Engine    — DAG execution, parallel nodes, checkpoint, streaming
├── Tool System         — built-in tools + MCP protocol + plugin extension
├── Context Engine      — LSP code understanding + semantic index + Git-aware
├── Sandbox             — macOS Seatbelt / Linux Bubblewrap (OS-level isolation)
├── Memory              — short-term / long-term / episodic + vector store
├── Permission          — tiered allow/deny/ask + audit trail
└── Session Manager     — persistence, checkpoint, branching, context compaction

Infrastructure
├── Scheduler           — Cron + async long-running agents
├── Plugin System       — Go / WASM / subprocess multi-form plugins
└── Telemetry           — OpenTelemetry observability
```

---

## Core Capabilities

### General Intelligence
- **Deep Research** — web search + multi-source verification + knowledge synthesis
- **Multi-Modal** — image understanding, document analysis, voice (planned)
- **Writing & Creation** — articles, reports, translation, editing, brainstorming
- **Office Productivity** — document processing, spreadsheet analysis, email drafting
- **Software Engineering** — full-stack code generation, refactoring, debugging, testing, deployment

### Agent Platform
- **Single Go Binary** — zero runtime dependencies, <100MB idle memory
- **Multi-Surface** — CLI, TUI, IDE plugin, Web Dashboard, IM Bot — one API
- **Multi-Model** — Anthropic Claude / OpenAI GPT / Google Gemini / DeepSeek / local (Ollama)
- **Graph-Based Orchestration** — ReAct, Plan-Execute, Multi-Agent, Human-in-the-Loop
- **Streaming Responses** — token-level real-time output, visible reasoning
- **Session Management** — persistent, checkpointed, branchable, auto-compacting

### Tools & Safety
- **Rich Tool Set** — file ops, shell exec, git, browser, web search, code intelligence
- **MCP Protocol** — Model Context Protocol, bidirectional (client + server)
- **Native Sandbox** — macOS Seatbelt / Linux Bubblewrap, OS-level process isolation
- **Permission System** — allow / deny / ask, policy engine, complete audit trail
- **LSP Integration** — Tree-sitter multi-language parsing + semantic embedding index

### Platform & Ecosystem
- **Multi-Channel Messaging** — Telegram, Discord, Slack, WhatsApp, WeChat bots
- **Scheduled Execution** — Cron jobs + async long-running agent tasks
- **Plugin Ecosystem** — WASM plugins + subprocess plugins, community-extensible
- **Audit Transparency** — every diff reviewable, complete operation logs

---

## Package Layout

```
baize/
├── cmd/baize/             # CLI / TUI entry point (main binary)
├── cmd/baized/            # Daemon / API Server
├── agent/                 # Agent abstraction (LLM, nodes, messages, state, structured output)
├── orchestrator/          # Agent orchestration (ReAct / Plan / Multi / HITL)
├── tool/                  # Tool system (built-in + MCP + plugin registry)
├── sandbox/               # OS sandbox (Seatbelt + Bubblewrap)
├── context/               # Project context engine (LSP + indexing + Git-aware)
├── session/               # Session management (persistence + checkpointing)
├── permission/            # Permission system (policy engine + audit)
├── memory/                # Memory system (short-term + long-term + vector store)
├── server/                # API Server (HTTP + WebSocket + gRPC)
├── tui/                   # Terminal UI (Bubble Tea framework)
├── plugin/                # Plugin system (WASM + subprocess)
├── gateway/               # Multi-channel IM gateway
├── scheduler/             # Job scheduler (Cron + async)
├── conf/                  # Configuration (TOML)
├── web/                   # Web Dashboard (React + TypeScript + Vite)
├── ide/                   # IDE plugins (VS Code + JetBrains)
├── examples/              # Example programs
└── docs/                  # Technical documentation
```

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
