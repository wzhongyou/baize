# 更新日志

## [0.4.0] - 2026-06-27

### 新增
- 全栈 AI 编码助手完整架构：引擎 + SDK + IDE 插件 + TUI
- CLI/TUI 独立交付（Bubble Tea 全屏终端界面）
- API Server（HTTP + SSE 流式输出）
- VSCode / JetBrains IDE 插件骨架
- MCP 协议客户端/服务端支持
- 权限引擎（策略审计、不可绕过）
- 会话持久化（SQLite WAL）
- 内置工具：文件 / Shell / Git / 网络 / 计算器

### 变更
- 图执行引擎从 Graphflow v0.2.0 升级至 Weave v0.4.0
- 目录重组，模块边界清晰（core / server / cli / ide / protocol）

---

## [0.3.0] - 2026-06-01

### 新增
- Agent 核心库：ReAct / RAG / Supervisor 模式
- LLM 网关（20+ 模型提供商）
- 结构化输出（JSON Schema + 校验）
- 短期/长期记忆接口
