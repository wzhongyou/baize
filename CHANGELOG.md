# 更新日志

## [0.4.0] - 2026-06-27

### 变更
- 升级 Graphflow v0.2.0 → Weave v0.4.0
- 目录重组，CLI/TUI 独立交付
- 移除竞品对标，聚焦 Baize 自身定位
- 清理 README_EN、Graphflow 引用残留

### 修复
- CI 构建流程（Graphflow 依赖克隆、gitignore 锚定）
- 文档与代码一致性

---

## [0.3.0] - 2026-06-06

### 新增
- 品牌更名：仓颉 → 白泽（Baize）
- 系统提示词重写为通用超级智能体
- Agent 核心库：ReAct / RAG / Supervisor 模式
- MCP 客户端、LLM 网关（多模型支持）
- 结构化输出（JSON Schema + 校验）
- 短期/长期记忆（向量存储）
- 内置工具：计算器 / 文件 / Shell / Git
- CLI/TUI 入口（Bubble Tea）、API Server（HTTP + SSE）
- 会话管理（持久化）、图执行引擎（Graphflow）
- 5 示例 + 40 单元测试
