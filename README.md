# Baize（白泽）

> 白泽，神兽也，达于万物，知其名、识其形、通其道。

Baize 是开源 AI 编程 Agent，在本地运行，帮你写代码、读代码、跑命令。

**CLI · TUI · VSCode · JetBrains**

---

## 为什么是 Baize

- **不绑定模型**：llmgate 接入 20+ 提供商（DeepSeek、Qwen、Kimi、OpenAI 等），一行配置切换
- **本地优先**：支持 Ollama / LM Studio 本地模型，数据不出机器
- **权限不可绕过**：工具执行前强制权限检查，模型无法越权操作文件系统
- **单二进制**：Go 编译，无运行时依赖，`curl | sh` 一行安装

---

## 快速开始

```bash
git clone https://github.com/wzhongyou/baize.git
cd baize
go build -o baize ./cli/

cp conf/llmgate.toml.example conf/llmgate.toml
# 填入 API key

./baize                      # 交互式 TUI
./baize "搜索登录相关代码"    # 单次执行
./baize server               # 启动 API Server（供 IDE 插件接入）
```

---

## 项目结构

```
baize/
├── cli/              CLI 入口 + Bubble Tea TUI
├── core/
│   ├── agent/        Agent 引擎（ReAct · Supervisor · RAG）
│   ├── tool/         工具系统（9 个内置 + MCP）
│   ├── skill/        Skill 能力包系统
│   ├── permission/   权限引擎
│   ├── session/      SQLite 会话持久化
│   ├── memory/       长期记忆
│   └── context/      项目指令 + Context 预算管理
├── server/           HTTP + SSE API Server
├── protocol/         智能体会话协议类型
└── docs/             技术设计文档
```

---

## 能力一览

| 能力 | 说明 |
|------|------|
| ReAct Agent | Weave 图编排引擎，思考→工具→循环 |
| Supervisor Agent | 多子 Agent 路由调度 |
| 9 个内置工具 | 文件、Grep、Shell、Git、Web 搜索/抓取、计算器、记忆、Skill |
| MCP 协议 | 动态扩展工具，双向集成 |
| Skill 系统 | 可安装能力包，两级加载，按需注入 context |
| 权限引擎 | suggest / auto-edit / full-auto 三档 |
| 会话持久化 | SQLite，多轮历史，支持恢复 |
| 长期记忆 | Markdown 文件存储，Agent 自动写入 |
| 多模型网关 | 20+ 提供商 + 本地模型，熔断重试 |
| 单二进制 | 零依赖，curl 一行安装 |

---

## 路线图

| 已完成 | 规划中 |
|--------|--------|
| ReAct + Supervisor Agent | 阶段化执行 + 验证门 |
| 9 工具 + MCP + Skill + 权限 | 代码库语义索引 |
| Bubble Tea TUI + API Server | 系统级沙箱（Seatbelt/Landlock）|
| 会话持久化 + 自动记忆 | VSCode / JetBrains 插件 |
| Context 预算 + 滚动压缩 | Eval 评测体系 |

---

## 生态仓库

| 项目 | 说明 |
|------|------|
| [Weave](https://github.com/wzhongyou/weave) | Go 图执行引擎 |
| [llmgate](https://github.com/wzhongyou/llmgate) | LLM 多模型网关 |
| [Carrel](https://github.com/wzhongyou/carrel) | AI Agent 安全沙箱 |

---

[MIT](LICENSE) © 2026 Wang Zhongyou
