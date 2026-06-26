# Baize（白泽）

> 白泽，神兽也，达于万物，知其名、识其形、通其道。

Baize 是开源的全栈 AI 编码助手。Go 引擎 + 多端客户端，单二进制分发。

**CLI · TUI · VSCode · JetBrains · Web Dashboard**

---

## 为什么是 Baize

Claude Code、Codex CLI 都绑定各自模型。Baize 通过 llmgate 接入 20+ 模型服务商，本地运行，数据不出机器。工具执行前有不可绕过的权限检查，模型无法越权。

---

## 快速开始

```bash
git clone https://github.com/wzhongyou/baize.git
cd baize
go build -o baize ./cmd/baize/

cp conf/llmgate.toml.example conf/llmgate.toml
# 编辑 conf/llmgate.toml 填入 API key

./baize                           # 交互式 TUI
./baize "搜索登录相关代码"          # 单次执行
./baize server                    # 启动 API 服务
```

### IDE 插件

| 插件 | 目录 | 状态 |
|------|------|:----:|
| VSCode | `ide/vscode/` | 本地可用 |
| JetBrains | `ide/jetbrains/` | 骨架就绪 |

---

## 项目结构

```
baize/
├── cmd/baize/          CLI 入口
├── agent/              Agent 引擎（ReAct · Supervisor · Graphflow）
├── tool/               工具系统（6 个内置 + MCP）
├── server/             HTTP + SSE API 服务
├── api/                API 协议类型
├── sdk/
│   ├── go/             Go SDK 客户端
│   └── ts/             TypeScript SDK（IDE 插件依赖）
├── session/            SQLite 会话持久化
├── permission/         权限引擎（已接入工具执行路径）
├── memory/             长期记忆
├── context/            项目文件分析
├── tui/                Bubble Tea 全屏终端
├── web/                Web Dashboard（React）
├── ide/
│   ├── vscode/         VSCode 插件
│   └── jetbrains/      JetBrains 插件
├── docs/               文档
└── examples/           示例
```

---

## 能力一览

| 能力 | |
|------|------|
| ReAct + Supervisor Agent | Graphflow 图编排 |
| Bubble Tea TUI | 全屏交互，流式渲染 |
| VSCode + JetBrains | IDE 深度集成 |
| Web Dashboard | React AGUI 界面 |
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
| Go SDK + TS SDK | LSP 深度代码智能 |
| VSCode + JetBrains 插件 | 系统级沙箱 |
| Web Dashboard | IM Bot 网关 |
| 会话 + 记忆 | Cron 定时任务 |

---

## 生态仓库

| 项目 | 说明 |
|------|------|
| [Weave](https://github.com/wzhongyou/weave) | Go 图执行引擎 |
| [llmgate](https://github.com/wzhongyou/llmgate) | LLM 多模型网关 |
| [Carrel](https://github.com/wzhongyou/carrel) | AI Agent 安全沙箱 |
| [Cangjie](https://github.com/wzhongyou/cangjie) | TypeScript CLI 学习项目 |

---

[MIT](LICENSE) © 2026 Wang Zhongyou
