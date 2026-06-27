# Changelog

All notable changes to Baize will be documented in this file.

## [Unreleased]

### Added
- System prompt rewritten for universal super-agent identity (was: software engineering assistant)
- English README with SEO-optimized keywords
- 15-dimension competitive analysis matrix (landscape survey)
- Short-term + long-term upgrade roadmap

### Changed
- Full brand rename: Cangjie → Baize (白泽)
- Binary: cj → baize
- CLI directory: cmd/cj/ → cmd/baize/
- Config path: .cangjie/ → .baize/
- Module path: github.com/wzhongyou/cangjie → github.com/wzhongyou/baize

### Fixed
- All 23 Go files import paths updated
- All documentation references renamed
- Binary naming consistency

---

## [0.3.0] - 2026-06-06

### Added
- Agent core library: ReAct / RAG / Supervisor Agent patterns
- MCP (Model Context Protocol) client implementation
- LLM gateway integration with multi-model support
- Structured output (JSON Schema) with validation
- Short-term memory + Long-term memory with Vector Store
- Calculator / File / Shell / Git built-in tools
- CLI/TUI entry point (Bubble Tea framework)
- API Server (HTTP + SSE)
- Session management with persistence
- Graph execution engine (via Graphflow)
- 5 example programs (agent_demo, streaming, supervisor, mcp, structured_output)
- 40 unit tests across agent/llmgate/permission/tool packages
