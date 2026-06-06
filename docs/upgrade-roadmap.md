# Baize（白泽）升级路线图

## 竞品差距总览

| 能力维度 | Manus | GenSpark | OpenClaw | Codex | CoWork | **Baize 当前** |
|---------|-------|----------|----------|-------|--------|----------------|
| **Web 浏览** | ✅ 内置 | ✅ 内置 | ❌ | ❌ | ❌ | ❌ |
| **沙箱执行** | ✅ | ✅ | ⚠️ 有限 | ✅ | ✅ | ❌ |
| **多 Agent 协作** | ⚠️ 单 Agent | ❌ | ❌ | ❌ | ✅ 核心 | ⚠️ 设计完成 |
| **IM 多渠道** | ❌ | ❌ | ✅ 核心 | ❌ | ❌ | ❌ |
| **代码智能** | ⚠️ 浅层 | ⚠️ 浅层 | ❌ | ✅ 深 | ⚠️ | ⚠️ LSP 接口 |
| **文档生成** | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| **数据分析** | ✅ | ✅ | ❌ | ❌ | ⚠️ | ❌ |
| **定时调度** | ❌ | ❌ | ⚠️ | ❌ | ❌ | ❌ |
| **插件系统** | ❌ | ❌ | ⚠️ | ❌ | ❌ | ❌ |
| **会话分支** | ❌ | ❌ | ❌ | ❌ | ❌ | ⚠️ 设计完成 |
| **权限系统** | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **MCP 协议** | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ 双向 |
| **Web Dashboard** | ❌ | ❌ | ❌ | ❌ | ❌ | ⚠️ MVP |
| **多模型支持** | ⚠️ 单模型 | ⚠️ 单模型 | ⚠️ | ⚠️ | ⚠️ | ✅ 多模型 |
| **IDE 插件** | ❌ | ❌ | ❌ | ⚠️ 有限 | ❌ | ❌ |

> 图例：✅ 具备 | ⚠️ 部分具备/设计完成 | ❌ 不具备

**核心差距**：Baize 的 Agent 核心引擎（Graphflow + ReAct + MCP）已经到位，但**工具层、安全层、分发层**需要补齐才能与竞品对位。

---

## 短期路线图（1-4 周）

> **目标**：让 Baize 成为可日常使用的 Agent 产品，补齐基础工具和安全能力。

### Week 1-2：工具系统 + 权限 + 会话

#### 1.1 工具系统填充
```
tool/builtin/
├── file.go          # [已启动] 文件读/写/编辑（支持 diff 预览）
├── shell.go         # [已启动] Shell 命令执行（超时控制 + 输出截断）
├── git.go           # [已启动] Git status/diff/log/branch/commit
├── web_search.go    # [新增] Web 搜索（Bing/Google/Brave API，多源结果）
├── web_fetch.go     # [新增] 网页内容抓取（URL → Markdown）
└── browser.go       # [新增] 无头浏览器（Playwright，用于 JS 页面）
```

#### 1.2 权限系统
```
permission/
├── permission.go    # [新增] 分级权限引擎（allow/deny/ask）
├── policy.go        # [新增] 策略引擎（基于 HCL/TOML 配置）
├── audit.go         # [新增] 审计日志（每步操作记录到 SQLite）
└── classifier.go    # [新增] 危险操作自动识别（rm -rf / curl | sh 等）
```

#### 1.3 会话持久化
```
session/
├── session.go       # [已有] 会话结构
├── store.go         # [新增] SQLite 持久化存储
├── checkpoint.go    # [新增] 检查点（支持回退）
├── branch.go        # [新增] 会话分支
└── compaction.go    # [新增] 上下文智能压缩
```

### Week 3-4：沙箱 + Web Dashboard + 上下文引擎

#### 2.1 OS 沙箱
```
sandbox/
├── seatbelt.go      # [新增] macOS Seatbelt（sandbox-exec）
├── bubblewrap.go    # [新增] Linux Bubblewrap（bwrap）
├── profile.go       # [新增] 沙箱配置（网络/文件系统/进程限制）
└── runner.go        # [新增] 统一执行接口
```

#### 2.2 Web Dashboard 完善
```
web/src/
├── App.tsx          # [已有] 主布局
├── components/
│   ├── ChatPanel.tsx       # [改进] 流式对话 + 消息管理
│   ├── SessionList.tsx     # [已有] 会话列表
│   ├── ToolCallLog.tsx     # [新增] 工具调用实时日志
│   ├── PermissionPrompt.tsx # [新增] 权限确认弹窗
│   └── DiffViewer.tsx      # [新增] 代码 Diff 查看器
└── hooks/
    └── useSSE.ts          # [新增] Server-Sent Events 流式连接
```

#### 2.3 上下文引擎
```
context/
├── project.go       # [已有] 项目分析器
├── lsp.go           # [新增] LSP 协议集成（符号跳转、引用查找、诊断）
├── index.go         # [新增] 代码索引（Tree-sitter AST + BM25 全文）
├── embed.go         # [新增] 向量嵌入（BGE-M3 / 本地模型）
└── retriever.go     # [新增] 混合检索器（BM25 + Vector + Rerank）
```

---

## 长期路线图（5-16 周）

> **目标**：全面超越单一竞品，构建最完整的开源 Agent 平台。

### Phase 3（Week 5-8）：平台化

#### 3.1 多渠道消息网关
```
gateway/
├── gateway.go       # [新增] 统一消息路由
├── telegram/        # [新增] Telegram Bot
├── discord/         # [新增] Discord Bot
├── slack/           # [新增] Slack Bot
├── whatsapp/        # [新增] WhatsApp Business API
└── wechat/          # [新增] 微信公众号/企业微信
```

**对标 OpenClaw**：Baize 的 IM Bot 不只见消息转发，而是每个 Bot 背后都是完整的 Agent，可以自主执行任务。

#### 3.2 调度系统
```
scheduler/
├── scheduler.go     # [新增] Cron 调度器
├── job.go           # [新增] 任务定义（一次性/周期性）
├── executor.go      # [新增] 异步 Agent 执行
├── retry.go         # [新增] 失败重试 + 指数退避
└── notifier.go      # [新增] 执行结果通知（Webhook/IM/Email）
```

**场景示例**：
```bash
baize job create --name "daily-code-review" \
  --cron "0 9 * * 1-5" \
  --prompt "Review all PRs opened in the last 24h and summarize"
```

#### 3.3 插件系统
```
plugin/
├── registry.go      # [新增] 插件注册表
├── wasm/            # [新增] WASM 插件运行时
├── process/         # [新增] 子进程插件
├── manifest.go      # [新增] 插件清单解析
└── store.go         # [新增] 插件市场（GitHub Releases）
```

### Phase 4（Week 9-12）：深度 & 生态

#### 4.1 深度研究模式
- **Multi-Source Verification**：对每个事实性声明，自动搜索 3+ 独立来源交叉验证
- **Knowledge Synthesis**：不是简单摘录，而是结构化整合（对比表、时间线、因果图）
- **Citation Tracking**：报告中每条结论附带来源 URL 和引用片段

#### 4.2 多 Agent 协作
```
orchestrator/
├── multi_agent.go   # [新增] Multi-Agent 编排
├── debate.go        # [新增] Agent 辩论/投票（用于关键决策）
├── pipeline.go      # [新增] Agent 流水线（A 产出 → B 审查 → C 执行）
└── supervisor.go    # [已有] Supervisor Agent（监督子 Agent 执行）
```

**场景**：
```
用户: "重构这个认证模块，确保安全"
  ├── Agent A (Explorer): 搜索所有认证相关代码
  ├── Agent B (Planner): 分析依赖，制定重构计划
  ├── Agent C (Executor): 执行重构（在 Git worktree 中）
  ├── Agent D (Reviewer): 安全审查重构结果
  └── Agent E (Verifier): 运行测试，验证功能完整性
```

#### 4.3 数据分析与可视化
- 支持 CSV/Excel/JSON 数据输入
- Python 运行时集成（sandbox 内执行 pandas/matplotlib）
- 自动生成图表（折线图、柱状图、热力图）
- 自然语言 → SQL → 数据库查询

#### 4.4 IDE 深度集成
```
ide/
├── vscode/          # [新增] VS Code 插件
│   ├── extension.ts       # 插件入口
│   ├── chat/              # Chat Panel（React Webview）
│   ├── inline/            # Inline Edit + Tab Completion
│   └── diff/              # Diff Review Panel
└── jetbrains/       # [新增] JetBrains 插件
```

### Phase 5（Week 13-16）：企业 & 社区

#### 5.1 企业特性
- **SSO 集成**（OIDC / SAML）
- **团队空间**（多用户协作、共享会话）
- **审计合规**（SOC 2 友好日志、数据保留策略）
- **私有部署**（Docker Compose / K8s Helm Chart）
- **用量计量**（Token 用量、Agent 执行耗时、计费接口）

#### 5.2 开源社区
- **贡献指南** + **开发者文档**
- **Example Library**（社区贡献的 Agent 模板）
- **Plugin Marketplace**（社区插件商店）
- **Benchmark Suite**（SWE-Bench / GAIA / WebArena 复现）
- **Weekly Office Hours**（社区维护者直播）

#### 5.3 多模态（探索）
- 图片输入（GPT-4V / Claude Vision）
- PDF/Screenshot → Markdown 解析
- 语音输入（Whisper 集成）
- 屏幕录制 → Agent 步骤回放

---

## 差异化策略

Baize 不做"又一个 Agent 工具"，而是构建三个竞品做不到的差异化：

### 1. 多入口统一 Agent
```
OpenClaw 只有 IM Bot
Codex 只有 IDE
Manus 只有 Web
GenSpark 只有 Web
CoWork 只有 API
Baize → CLI + TUI + IDE + Web + IM = 同一个 Agent，全平台
```

### 2. OS 原生沙箱 + 权限分级
```
竞品：云端沙箱（延迟高、代码上传风险）
Baize：本地 OS 沙箱（Seatbelt/Bubblewrap）+ 分级权限（独立于模型）
→ 安全且隐私，企业可审计
```

### 3. MCP 双向协议
```
竞品：不使用 MCP 或仅单向
Baize：既是 MCP Client（调用外部工具），也是 MCP Server（被其他 Agent 调用）
→ 可以嵌入任何 MCP 生态
```

---

## 版本里程碑

| 版本 | 时间 | 标志性能力 |
|------|------|-----------|
| **v0.3** | 🟢 当前 | ReAct Agent + MCP + Graphflow + Web Dashboard MVP |
| **v0.4** | +2 周 | 完整工具集 + 权限系统 + 会话持久化 |
| **v0.5** | +4 周 | OS 沙箱 + Web Dashboard 完善 + 上下文引擎 |
| **v0.6** | +8 周 | IM Bot 多渠道 + 调度系统 + 插件系统 |
| **v0.7** | +12 周 | Multi-Agent + 深度研究 + 数据分析 |
| **v1.0** | +16 周 | IDE 插件 + 企业特性 + 社区运营 |

---

*最后更新：2026-06-07*
