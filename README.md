# Baize（白泽）

> 白泽，神兽也，达于万物，知其名、识其形、通其道。

Baize 是开源的全栈 AI 编码助手。Go 引擎 + 多端客户端，单二进制分发。

**CLI · TUI · VSCode · JetBrains**

---

## 为什么是 Baize

Baize 不绑定任何单一模型服务商。通过 llmgate 接入 20+ 模型提供商，本地运行，数据不出机器。工具执行前有不可绕过的权限检查，模型无法越权。

---

## 快速开始

```bash
git clone https://github.com/wzhongyou/baize.git
cd baize
go build -o baize ./cli/

cp conf/llmgate.toml.example conf/llmgate.toml
# 编辑 conf/llmgate.toml 填入 API key

./baize                           # 交互式 TUI
./baize "搜索登录相关代码"          # 单次执行
./baize server                    # 启动 API 服务
```

---

## 项目结构

```
baize/
├── cli/                 CLI 入口 + TUI
│   └── tui/             Bubble Tea 全屏终端
├── core/                AI 引擎
│   ├── agent/           Agent 引擎（ReAct · Supervisor · RAG）
│   │   └── llmgate/     llmgate 适配器
│   ├── tool/            工具系统（6 个内置 + MCP）
│   │   ├── builtin/
│   │   └── mcp/
│   ├── permission/      权限引擎
│   ├── session/         SQLite 会话持久化
│   ├── memory/          长期记忆
│   └── context/         项目文件分析
├── server/              HTTP + SSE API 服务
│   └── middleware/
├── protocol/            API 协议类型（智能体会话协议）
├── ide/
│   ├── vscode/          VSCode 插件
│   └── jetbrains/       JetBrains 插件
├── docs/                文档
```

---

## 能力一览

| 能力 | |
|------|------|
| ReAct + Supervisor Agent | Weave 图编排引擎 |
| Bubble Tea TUI | 全屏交互，流式渲染 |
| VSCode + JetBrains | IDE 深度集成 |
| 6 个内置工具 | 文件、Shell、Git、Web 搜索/抓取、计算器 |
| MCP 协议 | 动态扩展工具 |
| 权限引擎 | 策略引擎接入 ToolNode，不可绕过 |
| 会话持久化 | SQLite，支持恢复 |
| 长期记忆 | 文件式 Markdown 存储 |
| 多模型 | llmgate 20+ 提供商 |
| 单二进制 | Go 编译，零依赖部署 |

---

## 路线图

| 已完成 | 规划中 |
|--------|--------|
| Agent Loop + TUI + API Server | Plan-Execute 模式 |
| 6 工具 + MCP + 权限 | Multi-Agent 辩论 |
| VSCode + JetBrains 插件 | 系统级沙箱 |
| LSP 深度代码智能 | 语言服务器协议 |
| 会话 + 记忆 | IM Bot 网关 |

---

## 生态仓库

| 项目 | 说明 |
|------|------|
| [Weave](https://github.com/wzhongyou/weave) | Go 图执行引擎 |
| [llmgate](https://github.com/wzhongyou/llmgate) | LLM 多模型网关 |
| [Carrel](https://github.com/wzhongyou/carrel) | AI Agent 安全沙箱 |


---

[MIT](LICENSE) © 2026 Wang Zhongyou
